package healthcheck

import (
	"context"
	"crypto/x509"
	"errors"
	"fmt"
	"sort"
	"strings"

	pb "github.com/linkerd/linkerd2/controller/gen/public"
	sm "github.com/linkerd/linkerd2/pkg/servicemirror"
	tsclient "github.com/servicemeshinterface/smi-sdk-go/pkg/gen/client/split/clientset/versioned"
	corev1 "k8s.io/api/core/v1"

	"github.com/linkerd/linkerd2/pkg/k8s"
	"github.com/linkerd/linkerd2/pkg/tls"
	v1 "k8s.io/api/rbac/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/tools/clientcmd"
)

const (
	// LinkerdMulticlusterSourceChecks adds a series of checks to validate
	// the source side of a multicluster setup
	LinkerdMulticlusterSourceChecks CategoryID = "linkerd-multicluster-source"

	// LinkerdMulticlusterTargetChecks add a series of checks to validate the
	// targetside of a multicluster setup
	LinkerdMulticlusterTargetChecks CategoryID = "linkerd-multicluster-target"

	linkerdServiceMirrorComponentName   = "linkerd-service-mirror"
	linkerdServiceMirrorClusterRoleName = "linkerd-service-mirror-access-local-resources"
	linkerdServiceMirrorRoleName        = "linkerd-service-mirror-read-remote-creds"
)

var expectedServiceMirrorClusterRolePolicies = []expectedPolicy{
	{
		resources: []string{"endpoints", "services"},
		verbs:     []string{"list", "get", "watch", "create", "delete", "update"},
	},
	{
		resources: []string{"namespaces"},
		verbs:     []string{"create", "list", "get", "watch"},
	},
}

var expectedServiceMirrorRolePolicies = []expectedPolicy{
	{
		resources: []string{"secrets"},
		verbs:     []string{"list", "get", "watch"},
	},
}

var expectedServiceMirrorRemoteClusterPolicyVerbs = []string{
	"get",
	"list",
	"watch",
}

func (hc *HealthChecker) multiClusterCategory() []category {
	return []category{
		{
			id: LinkerdMulticlusterSourceChecks,
			checkers: []checker{
				{
					description:   "service mirror controller is running",
					hintAnchor:    "l5d-multicluster-service-mirror-running",
					retryDeadline: hc.RetryDeadline,
					fatal:         true,
					check: func(context.Context) error {
						return hc.checkServiceMirrorController()
					},
				},
				{
					description: "service mirror controller ClusterRoles exist",
					hintAnchor:  "l5d-multicluster-cluster-role-exist",
					check: func(context.Context) error {
						if hc.Options.SourceCluster {
							return hc.checkClusterRoles(true, []string{linkerdServiceMirrorClusterRoleName}, hc.serviceMirrorComponentsSelector())
						}
						return &SkipError{Reason: "not checking muticluster"}
					},
				},
				{
					description: "service mirror controller ClusterRoleBindings exist",
					hintAnchor:  "l5d-multicluster-cluster-role-binding-exist",
					check: func(context.Context) error {
						if hc.Options.SourceCluster {
							return hc.checkClusterRoleBindings(true, []string{linkerdServiceMirrorClusterRoleName}, hc.serviceMirrorComponentsSelector())
						}
						return &SkipError{Reason: "not checking muticluster"}
					},
				},
				{
					description: "service mirror controller Roles exist",
					hintAnchor:  "l5d-multicluster-role-exist",
					check: func(context.Context) error {
						if hc.Options.SourceCluster {
							return hc.checkRoles(true, hc.serviceMirrorNs, []string{linkerdServiceMirrorRoleName}, hc.serviceMirrorComponentsSelector())
						}
						return &SkipError{Reason: "not checking muticluster"}
					},
				},
				{
					description: "service mirror controller RoleBindings exist",
					hintAnchor:  "l5d-multicluster-role-binding-exist",
					check: func(context.Context) error {
						if hc.Options.SourceCluster {
							return hc.checkRoleBindings(true, hc.serviceMirrorNs, []string{linkerdServiceMirrorRoleName}, hc.serviceMirrorComponentsSelector())
						}
						return &SkipError{Reason: "not checking muticluster"}
					},
				},
				{
					description: "service mirror controller ServiceAccounts exist",
					hintAnchor:  "l5d-multicluster-service-account-exist",
					check: func(context.Context) error {
						if hc.Options.SourceCluster {
							return hc.checkServiceAccounts([]string{linkerdServiceMirrorComponentName}, hc.serviceMirrorNs, hc.serviceMirrorComponentsSelector())
						}
						return &SkipError{Reason: "not checking muticluster"}
					},
				},
				{
					description: "service mirror controller has required permissions",
					hintAnchor:  "l5d-multicluster-source-rbac-correct",
					check: func(context.Context) error {
						return hc.checkServiceMirrorLocalRBAC()
					},
				},
				{
					description: "service mirror controller can access target clusters",
					hintAnchor:  "l5d-smc-target-clusters-access",
					check: func(context.Context) error {
						return hc.checkRemoteClusterConnectivity()
					},
				},
				{
					description: "all target cluster gateways are alive",
					hintAnchor:  "l5d-multicluster-target-gateways-alive",
					check: func(ctx context.Context) error {
						return hc.checkRemoteClusterGatewaysHealth(ctx)
					},
				},
				{
					description: "clusters share trust anchors",
					hintAnchor:  "l5d-multicluster-clusters-share-anchors",
					check: func(ctx context.Context) error {
						return hc.checkRemoteClusterAnchors()
					},
				},
				{
					description: "multicluster daisy chaining is avoided",
					hintAnchor:  "l5d-multicluster-daisy-chaining",
					warning:     true,
					check: func(ctx context.Context) error {
						return hc.checkDaisyChains()
					},
				},
				{
					description: "all mirror services have endpoints",
					hintAnchor:  "l5d-multicluster-services-endpoints",
					warning:     true,
					check: func(ctx context.Context) error {
						return hc.checkIfMirrorServicesHaveEndpoints()
					},
				},
				{
					description: "all gateway mirrors have endpoints",
					hintAnchor:  "l5d-multicluster-gateways-endpoints",
					warning:     true,
					check: func(ctx context.Context) error {
						return hc.checkIfGatewayMirrorsHaveEndpoints()
					},
				},
				{
					description: "remote: all referenced gateways are valid",
					hintAnchor:  "l5d-multicluster-gateways-exist",
					warning:     true,
					check: func(ctx context.Context) error {
						return hc.checkRemoteGateways()
					},
				},
			},
		},
		{
			id: LinkerdMulticlusterTargetChecks,
			checkers: []checker{
				{
					description: "all cluster gateways are valid",
					hintAnchor:  "l5d-multicluster-gateways-exist",
					warning:     true,
					check: func(ctx context.Context) error {
						targetCluster, err := hc.isTargetCluster()
						if err != nil {
							return err
						}
						if targetCluster || hc.TargetCluster {
							err := hc.checkLocalGateways()
							if err != nil {
								return err
							}
							return hc.checkIfGatewaysHaveEndpoints()
						}
						return &SkipError{Reason: "not checking target cluster"}
					},
				},
			},
		},
	}
}

func (hc *HealthChecker) serviceMirrorComponentsSelector() string {
	return fmt.Sprintf("%s=%s", k8s.ControllerComponentLabel, linkerdServiceMirrorComponentName)
}

func (hc *HealthChecker) checkServiceMirrorController() error {
	options := metav1.ListOptions{
		LabelSelector: hc.serviceMirrorComponentsSelector(),
	}
	result, err := hc.kubeAPI.AppsV1().Deployments(corev1.NamespaceAll).List(options)
	if err != nil {
		return err
	}

	// if we have explicitly requested for multicluster to be checked, error out
	if len(result.Items) == 0 && hc.Options.SourceCluster {
		return errors.New("Service mirror controller is not present")
	}

	if len(result.Items) > 0 {
		hc.Options.SourceCluster = true

		if len(result.Items) > 1 {
			var errors []error
			for _, smc := range result.Items {
				errors = append(errors, fmt.Errorf("%s/%s", smc.Namespace, smc.Name))
			}
			return fmt.Errorf("There are more than one service mirror controllers:\n%s", joinErrors(errors, 1))
		}

		controller := result.Items[0]
		if controller.Status.AvailableReplicas < 1 {
			return fmt.Errorf("Service mirror controller is not available: %s/%s", controller.Namespace, controller.Name)
		}
		hc.serviceMirrorNs = controller.Namespace
		return nil
	}

	return &SkipError{Reason: "not checking muticluster"}
}

func comparePermissions(expected, actual []string) error {
	sort.Strings(expected)
	sort.Strings(actual)

	expectedStr := strings.Join(expected, ",")
	actualStr := strings.Join(actual, ",")

	if expectedStr != actualStr {
		return fmt.Errorf("expected %s, got %s", expectedStr, actualStr)
	}

	return nil
}

func verifyRule(expected expectedPolicy, actual []v1.PolicyRule) error {
	for _, rule := range actual {
		if err := comparePermissions(expected.resources, rule.Resources); err == nil {
			if err := comparePermissions(expected.verbs, rule.Verbs); err != nil {
				return fmt.Errorf("unexpected verbs %s", err)
			}
			return nil
		}
	}
	return fmt.Errorf("could not fine rule for %s", strings.Join(expected.resources, ","))
}

func (hc *HealthChecker) checkServiceMirrorLocalRBAC() error {
	if hc.Options.SourceCluster {
		var errors []string

		clusterRole, err := hc.kubeAPI.RbacV1().ClusterRoles().Get(linkerdServiceMirrorClusterRoleName, metav1.GetOptions{})
		if err != nil {
			return fmt.Errorf("Could not obtain service mirror ClusterRole %s: %s", linkerdServiceMirrorClusterRoleName, err)
		}

		role, err := hc.kubeAPI.RbacV1().Roles(hc.serviceMirrorNs).Get(linkerdServiceMirrorRoleName, metav1.GetOptions{})
		if err != nil {
			return fmt.Errorf("Could not obtain service mirror Role %s : %s", linkerdServiceMirrorRoleName, err)
		}

		if len(clusterRole.Rules) != len(expectedServiceMirrorClusterRolePolicies) {
			return fmt.Errorf("Service mirror ClusterRole %s has %d policy rules, expected %d", clusterRole.Name, len(clusterRole.Rules), len(expectedServiceMirrorClusterRolePolicies))
		}

		for _, rule := range expectedServiceMirrorClusterRolePolicies {
			if err := verifyRule(rule, clusterRole.Rules); err != nil {
				errors = append(errors, fmt.Sprintf("Service mirror ClusterRole %s: %s", clusterRole.Name, err))
			}
		}

		if len(role.Rules) != len(expectedServiceMirrorRolePolicies) {
			return fmt.Errorf("Service mirror Role %s has %d policy rules, expected %d", role.Name, len(role.Rules), len(expectedServiceMirrorRolePolicies))
		}

		for _, rule := range expectedServiceMirrorRolePolicies {
			if err := verifyRule(rule, role.Rules); err != nil {
				errors = append(errors, fmt.Sprintf("Service mirror Role %s: %s", role.Name, err))
			}
		}

		if len(errors) > 0 {
			return fmt.Errorf(strings.Join(errors, "\n"))
		}

		return nil
	}
	return &SkipError{Reason: "not checking muticluster"}
}

func (hc *HealthChecker) checkRemoteClusterAnchors() error {
	if len(hc.remoteClusterConfigs) == 0 {
		return &SkipError{Reason: "no target cluster configs"}
	}

	localAnchors, err := tls.DecodePEMCertificates(hc.linkerdConfig.Global.IdentityContext.TrustAnchorsPem)
	if err != nil {
		return fmt.Errorf("Cannot parse source trust anchors: %s", err)
	}

	var offendingClusters []string
	for _, cfg := range hc.remoteClusterConfigs {

		clientConfig, err := clientcmd.RESTConfigFromKubeConfig(cfg.APIConfig)
		if err != nil {
			offendingClusters = append(offendingClusters, fmt.Sprintf("* %s: unable to parse api config", cfg.ClusterName))
			continue
		}

		remoteAPI, err := k8s.NewAPIForConfig(clientConfig, "", []string{}, requestTimeout)
		if err != nil {
			offendingClusters = append(offendingClusters, fmt.Sprintf("* %s: unable to instantiate api", cfg.ClusterName))
			continue
		}

		_, cfMap, err := FetchLinkerdConfigMap(remoteAPI, cfg.LinkerdNamespace)
		if err != nil {
			offendingClusters = append(offendingClusters, fmt.Sprintf("* %s: unable to fetch anchors: %s", cfg.ClusterName, err))
			continue
		}
		remoteAnchors, err := tls.DecodePEMCertificates(cfMap.Global.IdentityContext.TrustAnchorsPem)
		if err != nil {
			offendingClusters = append(offendingClusters, fmt.Sprintf("* %s: cannot parse trust anchors", cfg.ClusterName))
			continue
		}

		// we fail early if the lens are not the same. If they are the
		// same, we can only compare certs one way and be sure we have
		// identical anchors
		if len(remoteAnchors) != len(localAnchors) {
			offendingClusters = append(offendingClusters, fmt.Sprintf("* %s", cfg.ClusterName))
			continue
		}

		localAnchorsMap := make(map[string]*x509.Certificate)
		for _, c := range localAnchors {
			localAnchorsMap[string(c.Signature)] = c
		}

		for _, remote := range remoteAnchors {
			local, ok := localAnchorsMap[string(remote.Signature)]
			if !ok || !local.Equal(remote) {
				offendingClusters = append(offendingClusters, fmt.Sprintf("* %s", cfg.ClusterName))
				break
			}
		}
	}

	if len(offendingClusters) > 0 {
		return fmt.Errorf("Problematic clusters:\n    %s", strings.Join(offendingClusters, "\n    "))
	}

	return nil
}

func serviceExported(svc corev1.Service) bool {
	_, hasGtwName := svc.Annotations[k8s.GatewayNameAnnotation]
	_, hasGtwNs := svc.Annotations[k8s.GatewayNsAnnotation]
	return hasGtwName && hasGtwNs
}

func (hc *HealthChecker) checkDaisyChains() error {
	if hc.Options.SourceCluster {
		errs := []error{}

		svcs, err := hc.kubeAPI.CoreV1().Services(metav1.NamespaceAll).List(metav1.ListOptions{})
		if err != nil {
			return err
		}
		for _, svc := range svcs.Items {
			_, isMirror := svc.Labels[k8s.MirroredResourceLabel]
			if isMirror && serviceExported(svc) {
				errs = append(errs, fmt.Errorf("mirror service %s.%s is exported", svc.Name, svc.Namespace))
			}
		}

		ts, err := tsclient.NewForConfig(hc.kubeAPI.Config)
		if err != nil {
			return err
		}
		splits, err := ts.SplitV1alpha1().TrafficSplits(metav1.NamespaceAll).List(metav1.ListOptions{})
		if err != nil {
			return err
		}
		for _, split := range splits.Items {
			apex, err := hc.kubeAPI.CoreV1().Services(split.Namespace).Get(split.Spec.Service, metav1.GetOptions{})
			if err != nil {
				return err
			}
			if serviceExported(*apex) {
				for _, backend := range split.Spec.Backends {
					if backend.Weight.IsZero() {
						continue
					}
					leaf, err := hc.kubeAPI.CoreV1().Services(split.Namespace).Get(backend.Service, metav1.GetOptions{})
					if err != nil {
						return err
					}
					_, isMirror := leaf.Labels[k8s.MirroredResourceLabel]
					if isMirror {
						errs = append(errs, fmt.Errorf("exported service %s.%s routes to mirror service %s.%s via traffic split %s.%s",
							apex.Name, apex.Namespace, leaf.Name, leaf.Namespace, split.Name, split.Namespace,
						))
					}
				}
			}
		}
		if len(errs) > 0 {
			messages := []string{}
			for _, err := range errs {
				messages = append(messages, fmt.Sprintf("* %s", err.Error()))
			}
			return errors.New(strings.Join(messages, "\n"))
		}
		return nil
	}
	return &SkipError{Reason: "not checking muticluster"}
}

func (hc *HealthChecker) checkLocalGateways() error {

	errs := checkGateways(hc.kubeAPI)
	if len(errs) > 0 {
		return joinErrors(errs, 1)
	}
	return nil
}

func (hc *HealthChecker) checkRemoteGateways() error {

	if len(hc.remoteClusterConfigs) == 0 {
		return &SkipError{Reason: "no target cluster configs"}
	}

	var offendingClusters []error
	for _, cfg := range hc.remoteClusterConfigs {
		clientConfig, err := clientcmd.RESTConfigFromKubeConfig(cfg.APIConfig)
		if err != nil {
			offendingClusters = append(offendingClusters, fmt.Errorf("* %s: unable to parse api config", cfg.ClusterName))
			continue
		}

		remoteAPI, err := k8s.NewAPIForConfig(clientConfig, "", []string{}, requestTimeout)
		if err != nil {
			offendingClusters = append(offendingClusters, fmt.Errorf("* %s: unable to instantiate api", cfg.ClusterName))
			continue
		}

		errs := checkGateways(remoteAPI)
		if len(errs) > 0 {
			offendingClusters = append(offendingClusters, fmt.Errorf("* %s: remote cluster has invalid gateways:\n%s", cfg.ClusterName, joinErrors(errs, 2).Error()))
			continue
		}
	}
	if len(offendingClusters) > 0 {
		return joinErrors(offendingClusters, 1)
	}
	return nil
}

func checkGateways(api *k8s.KubernetesAPI) []error {
	errs := []error{}
	services, err := api.CoreV1().Services(metav1.NamespaceAll).List(metav1.ListOptions{})
	if err != nil {
		return []error{err}
	}

	for _, svc := range services.Items {
		if serviceExported(svc) {
			// Check if there is a relevant gateway
			gatewayName := svc.Annotations[k8s.GatewayNameAnnotation]
			gatewayNamespace := svc.Annotations[k8s.GatewayNsAnnotation]
			gateway, err := api.CoreV1().Services(gatewayNamespace).Get(gatewayName, metav1.GetOptions{})
			if err != nil {
				errs = append(errs, fmt.Errorf("Exported service %s.%s references a gateway that does not exist: %s.%s", svc.Name, svc.Namespace, gatewayName, gatewayNamespace))
				continue
			}

			// check if there is an external IP for the gateway service
			if len(gateway.Status.LoadBalancer.Ingress) <= 0 {
				errs = append(errs, fmt.Errorf("Exported service %s.%s references a gateway with no external IP: %s.%s", svc.Name, svc.Namespace, gatewayName, gatewayNamespace))
			}

			// check if the gateway service has relevant ports
			portNames := []string{k8s.GatewayPortName, k8s.ProbePortName}
			for _, portName := range portNames {
				if !ifPortExists(gateway.Spec.Ports, portName) {
					errs = append(errs, fmt.Errorf("Exported service %s.%s references a gateway that is missing port %s: %s.%s", svc.Name, svc.Namespace, portName, gatewayName, gatewayNamespace))
				}
			}

		}
	}
	return errs
}

func ifPortExists(ports []corev1.ServicePort, portName string) bool {
	for _, port := range ports {
		if port.Name == portName {
			return true
		}
	}
	return false
}

func (hc *HealthChecker) isTargetCluster() (bool, error) {

	services, err := hc.kubeAPI.CoreV1().Services(metav1.NamespaceAll).List(metav1.ListOptions{})
	if err != nil {
		return false, err
	}

	for _, service := range services.Items {
		if serviceExported(service) {
			return true, nil
		}
	}

	return false, nil
}

func (hc *HealthChecker) checkRemoteClusterConnectivity() error {
	if hc.Options.SourceCluster {
		options := metav1.ListOptions{
			FieldSelector: fmt.Sprintf("%s=%s", "type", k8s.MirrorSecretType),
		}
		secrets, err := hc.kubeAPI.CoreV1().Secrets(corev1.NamespaceAll).List(options)
		if err != nil {
			return err
		}

		if len(secrets.Items) == 0 {
			return &SkipError{Reason: "no target cluster configs"}
		}

		var errors []error
		for _, s := range secrets.Items {
			secret := s
			config, err := sm.ParseRemoteClusterSecret(&secret)
			if err != nil {
				errors = append(errors, fmt.Errorf("* secret: [%s/%s]: could not parse config secret: %s", secret.Namespace, secret.Name, err))
				continue
			}

			clientConfig, err := clientcmd.RESTConfigFromKubeConfig(config.APIConfig)
			if err != nil {
				errors = append(errors, fmt.Errorf("* secret: [%s/%s] cluster: [%s]: unable to parse api config: %s", secret.Namespace, secret.Name, config.ClusterName, err))
				continue
			}

			remoteAPI, err := k8s.NewAPIForConfig(clientConfig, "", []string{}, requestTimeout)
			if err != nil {
				errors = append(errors, fmt.Errorf("* secret: [%s/%s] cluster: [%s]: could not instantiate api for target cluster: %s", secret.Namespace, secret.Name, config.ClusterName, err))
				continue
			}

			var verbs []string
			if err := hc.checkCanPerformAction(remoteAPI, "get", corev1.NamespaceAll, "", "v1", "services"); err == nil {
				verbs = append(verbs, "get")
			}

			if err := hc.checkCanPerformAction(remoteAPI, "list", corev1.NamespaceAll, "", "v1", "services"); err == nil {
				verbs = append(verbs, "list")
			}

			if err := hc.checkCanPerformAction(remoteAPI, "watch", corev1.NamespaceAll, "", "v1", "services"); err == nil {
				verbs = append(verbs, "watch")
			}

			if err := comparePermissions(expectedServiceMirrorRemoteClusterPolicyVerbs, verbs); err != nil {
				errors = append(errors, fmt.Errorf("* cluster: [%s]: Insufficient Service permissions: %s", config.ClusterName, err))
			}

			hc.remoteClusterConfigs = append(hc.remoteClusterConfigs, config)

		}

		if len(errors) > 0 {
			return joinErrors(errors, 2)
		}
		return nil
	}
	return &SkipError{Reason: "not checking muticluster"}
}

func (hc *HealthChecker) checkRemoteClusterGatewaysHealth(ctx context.Context) error {
	if hc.Options.SourceCluster {
		if hc.apiClient == nil {
			return errors.New("public api client uninitialized")
		}
		req := &pb.GatewaysRequest{
			TimeWindow: "1m",
		}
		rsp, err := hc.apiClient.Gateways(ctx, req)
		if err != nil {
			return err
		}

		var deadGateways []string
		var aliveGateways []string
		if len(rsp.GetOk().GatewaysTable.Rows) == 0 {
			return &SkipError{Reason: "no target gateways"}
		}
		for _, gtw := range rsp.GetOk().GatewaysTable.Rows {
			if gtw.Alive {
				aliveGateways = append(aliveGateways, fmt.Sprintf("    * cluster: [%s], gateway: [%s/%s]", gtw.ClusterName, gtw.Namespace, gtw.Name))
			} else {
				deadGateways = append(deadGateways, fmt.Sprintf("* cluster: [%s], gateway: [%s/%s]", gtw.ClusterName, gtw.Namespace, gtw.Name))
			}
		}

		if len(deadGateways) > 0 {
			return fmt.Errorf("Some gateways are not alive:\n    %s", strings.Join(deadGateways, "\n    "))
		}
		return &VerboseSuccess{Message: strings.Join(aliveGateways, "\n")}
	}
	return &SkipError{Reason: "not checking muticluster"}
}

func (hc *HealthChecker) checkIfMirrorServicesHaveEndpoints() error {
	if hc.Options.SourceCluster {

		var servicesWithNoEndpoints []string
		selector := fmt.Sprintf("%s, !%s", k8s.MirroredResourceLabel, k8s.MirroredGatewayLabel)
		mirrorServices, err := hc.kubeAPI.CoreV1().Services(metav1.NamespaceAll).List(metav1.ListOptions{LabelSelector: selector})
		if err != nil {
			return err
		}

		for _, svc := range mirrorServices.Items {
			// Check if there is a relevant end-point
			endpoint, err := hc.kubeAPI.CoreV1().Endpoints(svc.Namespace).Get(svc.Name, metav1.GetOptions{})
			if err != nil || len(endpoint.Subsets) == 0 {
				servicesWithNoEndpoints = append(servicesWithNoEndpoints, fmt.Sprintf("%s.%s mirrored from cluster [%s] (gateway: [%s/%s])", svc.Name, svc.Namespace, svc.Labels[k8s.RemoteClusterNameLabel], svc.Labels[k8s.RemoteGatewayNsLabel], svc.Labels[k8s.RemoteGatewayNameLabel]))
			}
		}

		if len(servicesWithNoEndpoints) > 0 {
			return fmt.Errorf("Some mirror services do not have endpoints:\n    %s", strings.Join(servicesWithNoEndpoints, "\n    "))
		}
		return nil
	}
	return &SkipError{Reason: "not checking muticluster"}
}

func (hc *HealthChecker) checkIfGatewayMirrorsHaveEndpoints() error {
	if hc.Options.SourceCluster {

		var gatewayMirrorsWithNoEndpoints []string
		gatewayServices, err := hc.kubeAPI.CoreV1().Services(metav1.NamespaceAll).List(metav1.ListOptions{LabelSelector: k8s.MirroredGatewayLabel})
		if err != nil {
			return err
		}

		for _, svc := range gatewayServices.Items {
			// Check if there is a relevant end-point
			endpoints, err := hc.kubeAPI.CoreV1().Endpoints(svc.Namespace).Get(svc.Name, metav1.GetOptions{})
			if err != nil || len(endpoints.Subsets) == 0 {
				gatewayMirrorsWithNoEndpoints = append(gatewayMirrorsWithNoEndpoints, fmt.Sprintf("%s.%s mirrored from cluster [%s]", svc.Name, svc.Namespace, svc.Labels[k8s.RemoteClusterNameLabel]))
			}
		}

		if len(gatewayMirrorsWithNoEndpoints) > 0 {
			return fmt.Errorf("Some gateway mirrors do not have endpoints:\n    %s", strings.Join(gatewayMirrorsWithNoEndpoints, "\n    "))
		}
		return nil
	}
	return &SkipError{Reason: "not checking muticluster"}
}

func (hc *HealthChecker) checkIfGatewaysHaveEndpoints() error {

	var gatewaysWithNoEndpoints []string
	services, err := hc.kubeAPI.CoreV1().Services(corev1.NamespaceAll).List(metav1.ListOptions{})
	if err != nil {
		return err
	}

	for _, service := range services.Items {
		if gatewayService(service) {
			// Check if there is a relevant end-point
			endpoints, err := hc.kubeAPI.CoreV1().Endpoints(service.Namespace).Get(service.Name, metav1.GetOptions{})
			if err != nil || len(endpoints.Subsets) == 0 {
				gatewaysWithNoEndpoints = append(gatewaysWithNoEndpoints, fmt.Sprintf("%s.%s", service.Name, service.Namespace))
			}
		}

	}

	if len(gatewaysWithNoEndpoints) > 0 {
		return fmt.Errorf("Some gateway services do not have endpoints:\n    %s", strings.Join(gatewaysWithNoEndpoints, "\n    "))
	}
	return nil

}

func gatewayService(svc corev1.Service) bool {
	_, isGtw := svc.Annotations[k8s.MulticlusterGatewayAnnotation]
	return isGtw
}

func joinErrors(errs []error, tabDepth int) error {
	indent := strings.Repeat("    ", tabDepth)
	errStrings := []string{}
	for _, err := range errs {
		errStrings = append(errStrings, indent+err.Error())
	}
	return errors.New(strings.Join(errStrings, "\n"))
}
