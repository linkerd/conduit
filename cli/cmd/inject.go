package cmd

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"

	"github.com/ghodss/yaml"
	"github.com/linkerd/linkerd2/pkg/healthcheck"
	"github.com/linkerd/linkerd2/pkg/k8s"
	"github.com/spf13/cobra"
	appsV1 "k8s.io/api/apps/v1"
	batchV1 "k8s.io/api/batch/v1"
	"k8s.io/api/core/v1"
	"k8s.io/api/extensions/v1beta1"
	k8sMeta "k8s.io/apimachinery/pkg/api/meta"
	k8sResource "k8s.io/apimachinery/pkg/api/resource"
	metaV1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/intstr"
	yamlDecoder "k8s.io/apimachinery/pkg/util/yaml"
)

const (
	// LocalhostDNSNameOverride allows override of the controlPlaneDNS. This
	// must be in absolute form for the proxy to special-case it.
	LocalhostDNSNameOverride = "localhost."
	// ControlPlanePodName default control plane pod name.
	ControlPlanePodName = "linkerd-controller"
	// PodNamespaceEnvVarName is the name of the variable used to pass the pod's namespace.
	PodNamespaceEnvVarName = "LINKERD2_PROXY_POD_NAMESPACE"

	// for inject reports
	hostNetworkDesc = "hostNetwork: pods do not use host networking"
	sidecarDesc     = "sidecar: pods do not have a proxy or initContainer already injected"
	unsupportedDesc = "supported: at least one resource injected"
	udpDesc         = "udp: pod specs do not include UDP ports"
)

type injectOptions struct {
	*proxyConfigOptions
}

type resourceTransformerInject struct{}

// InjectYAML processes resource definitions and outputs them after injection in out
var InjectYAML = func(in io.Reader, out io.Writer, report io.Writer, options *injectOptions) error {
	return ProcessYAML(in, out, report, options, resourceTransformerInject{})
}

var runInjectCmd = func(inputs []io.Reader, errWriter, outWriter io.Writer, options *injectOptions) int {
	return transformInput(inputs, errWriter, outWriter, options, resourceTransformerInject{})
}

// objMeta provides a generic struct to parse the names of Kubernetes objects
type objMeta struct {
	metaV1.ObjectMeta `json:"metadata,omitempty" protobuf:"bytes,1,opt,name=metadata"`
}

func newInjectOptions() *injectOptions {
	return &injectOptions{
		proxyConfigOptions: newProxyConfigOptions(),
	}
}

func newCmdInject() *cobra.Command {
	options := newInjectOptions()

	cmd := &cobra.Command{
		Use:   "inject [flags] CONFIG-FILE",
		Short: "Add the Linkerd proxy to a Kubernetes config",
		Long: `Add the Linkerd proxy to a Kubernetes config.

You can use a config file from stdin by using the '-' argument
with 'linkerd inject'. e.g. curl http://url.to/yml | linkerd inject -
Also works with a folder containing resource files and other
sub-folder. e.g. linkerd inject <folder> | kubectl apply -f -
	`,
		RunE: func(cmd *cobra.Command, args []string) error {

			if len(args) < 1 {
				return fmt.Errorf("please specify a kubernetes resource file")
			}

			if err := options.validate(); err != nil {
				return err
			}

			in, err := read(args[0])
			if err != nil {
				return err
			}

			exitCode := runInjectCmd(in, os.Stderr, os.Stdout, options)
			os.Exit(exitCode)
			return nil
		},
	}

	addProxyConfigFlags(cmd, options.proxyConfigOptions)
	return cmd
}

// Returns the integer representation of os.Exit code; 0 on success and 1 on failure.
// TODO: move to separate file
func transformInput(inputs []io.Reader, errWriter, outWriter io.Writer, options *injectOptions, rt resourceTransformer) int {
	postInjectBuf := &bytes.Buffer{}
	reportBuf := &bytes.Buffer{}

	for _, input := range inputs {
		err := ProcessYAML(input, postInjectBuf, reportBuf, options, rt)
		if err != nil {
			// TODO: have a more generic error (shared with uninject)
			fmt.Fprintf(errWriter, "Error injecting linkerd proxy: %v\n", err)
			return 1
		}
		_, err = io.Copy(outWriter, postInjectBuf)

		// print error report after yaml output, for better visibility
		io.Copy(errWriter, reportBuf)

		if err != nil {
			fmt.Fprintf(errWriter, "Error printing YAML: %v\n", err)
			return 1
		}
	}
	return 0
}

/* Given a ObjectMeta, update ObjectMeta in place with the new labels and
 * annotations.
 */
func injectObjectMeta(t *metaV1.ObjectMeta, k8sLabels map[string]string, options *injectOptions) {
	if t.Annotations == nil {
		t.Annotations = make(map[string]string)
	}
	t.Annotations[k8s.CreatedByAnnotation] = k8s.CreatedByAnnotationValue()
	t.Annotations[k8s.ProxyVersionAnnotation] = options.linkerdVersion

	if t.Labels == nil {
		t.Labels = make(map[string]string)
	}
	t.Labels[k8s.ControllerNSLabel] = controlPlaneNamespace
	for k, v := range k8sLabels {
		t.Labels[k] = v
	}
}

/* Given a PodSpec, update the PodSpec in place with the sidecar
 * and init-container injected. If the pod is unsuitable for having them
 * injected, return false.
 */
func injectPodSpec(t *v1.PodSpec, identity k8s.TLSIdentity, controlPlaneDNSNameOverride string, options *injectOptions, report *injectReport) bool {
	report.hostNetwork = t.HostNetwork
	report.sidecar = healthcheck.HasExistingSidecars(t)
	report.udp = checkUDPPorts(t)

	// Skip injection if:
	// 1) Pods with `hostNetwork: true` share a network namespace with the host.
	//    The init-container would destroy the iptables configuration on the host.
	// OR
	// 2) Known sidecars already present.
	if report.hostNetwork || report.sidecar {
		return false
	}

	f := false
	inboundSkipPorts := append(options.ignoreInboundPorts, options.proxyControlPort, options.proxyMetricsPort)
	inboundSkipPortsStr := make([]string, len(inboundSkipPorts))
	for i, p := range inboundSkipPorts {
		inboundSkipPortsStr[i] = strconv.Itoa(int(p))
	}

	outboundSkipPortsStr := make([]string, len(options.ignoreOutboundPorts))
	for i, p := range options.ignoreOutboundPorts {
		outboundSkipPortsStr[i] = strconv.Itoa(int(p))
	}

	initArgs := []string{
		"--incoming-proxy-port", fmt.Sprintf("%d", options.inboundPort),
		"--outgoing-proxy-port", fmt.Sprintf("%d", options.outboundPort),
		"--proxy-uid", fmt.Sprintf("%d", options.proxyUID),
	}

	if len(inboundSkipPortsStr) > 0 {
		initArgs = append(initArgs, "--inbound-ports-to-ignore")
		initArgs = append(initArgs, strings.Join(inboundSkipPortsStr, ","))
	}

	if len(outboundSkipPortsStr) > 0 {
		initArgs = append(initArgs, "--outbound-ports-to-ignore")
		initArgs = append(initArgs, strings.Join(outboundSkipPortsStr, ","))
	}

	nonRoot := false
	runAsUser := int64(0)
	initContainer := v1.Container{
		Name:                     k8s.InitContainerName,
		Image:                    options.taggedProxyInitImage(),
		ImagePullPolicy:          v1.PullPolicy(options.imagePullPolicy),
		TerminationMessagePolicy: v1.TerminationMessageFallbackToLogsOnError,
		Args:                     initArgs,
		SecurityContext: &v1.SecurityContext{
			Capabilities: &v1.Capabilities{
				Add: []v1.Capability{v1.Capability("NET_ADMIN")},
			},
			Privileged:   &f,
			RunAsNonRoot: &nonRoot,
			RunAsUser:    &runAsUser,
		},
	}
	controlPlaneDNS := fmt.Sprintf("linkerd-proxy-api.%s.svc.cluster.local", controlPlaneNamespace)
	if controlPlaneDNSNameOverride != "" {
		controlPlaneDNS = controlPlaneDNSNameOverride
	}

	metricsPort := intstr.IntOrString{
		IntVal: int32(options.proxyMetricsPort),
	}

	proxyProbe := v1.Probe{
		Handler: v1.Handler{
			HTTPGet: &v1.HTTPGetAction{
				Path: "/metrics",
				Port: metricsPort,
			},
		},
		InitialDelaySeconds: 10,
	}

	resources := v1.ResourceRequirements{
		Requests: v1.ResourceList{},
	}

	if options.proxyCPURequest != "" {
		resources.Requests["cpu"] = k8sResource.MustParse(options.proxyCPURequest)
	}

	if options.proxyMemoryRequest != "" {
		resources.Requests["memory"] = k8sResource.MustParse(options.proxyMemoryRequest)
	}

	profileSuffixes := "."
	if options.disableExternalProfiles {
		profileSuffixes = "svc.cluster.local."
	}
	sidecar := v1.Container{
		Name:                     k8s.ProxyContainerName,
		Image:                    options.taggedProxyImage(),
		ImagePullPolicy:          v1.PullPolicy(options.imagePullPolicy),
		TerminationMessagePolicy: v1.TerminationMessageFallbackToLogsOnError,
		SecurityContext: &v1.SecurityContext{
			RunAsUser: &options.proxyUID,
		},
		Ports: []v1.ContainerPort{
			{
				Name:          "linkerd-proxy",
				ContainerPort: int32(options.inboundPort),
			},
			{
				Name:          "linkerd-metrics",
				ContainerPort: int32(options.proxyMetricsPort),
			},
		},
		Resources: resources,
		Env: []v1.EnvVar{
			{Name: "LINKERD2_PROXY_LOG", Value: options.proxyLogLevel},
			{Name: "LINKERD2_PROXY_BIND_TIMEOUT", Value: options.proxyBindTimeout},
			{
				Name:  "LINKERD2_PROXY_CONTROL_URL",
				Value: fmt.Sprintf("tcp://%s:%d", controlPlaneDNS, options.proxyAPIPort),
			},
			{Name: "LINKERD2_PROXY_CONTROL_LISTENER", Value: fmt.Sprintf("tcp://0.0.0.0:%d", options.proxyControlPort)},
			{Name: "LINKERD2_PROXY_METRICS_LISTENER", Value: fmt.Sprintf("tcp://0.0.0.0:%d", options.proxyMetricsPort)},
			{Name: "LINKERD2_PROXY_OUTBOUND_LISTENER", Value: fmt.Sprintf("tcp://127.0.0.1:%d", options.outboundPort)},
			{Name: "LINKERD2_PROXY_INBOUND_LISTENER", Value: fmt.Sprintf("tcp://0.0.0.0:%d", options.inboundPort)},
			{Name: "LINKERD2_PROXY_DESTINATION_PROFILE_SUFFIXES", Value: profileSuffixes},
			{
				Name:      PodNamespaceEnvVarName,
				ValueFrom: &v1.EnvVarSource{FieldRef: &v1.ObjectFieldSelector{FieldPath: "metadata.namespace"}},
			},
		},
		LivenessProbe:  &proxyProbe,
		ReadinessProbe: &proxyProbe,
	}

	// Special case if the caller specifies that
	// LINKERD2_PROXY_OUTBOUND_ROUTER_CAPACITY be set on the pod.
	// We key off of any container image in the pod. Ideally we would instead key
	// off of something at the top-level of the PodSpec, but there is nothing
	// easily identifiable at that level.
	// This is currently only used by the Prometheus pod in the control-plane.
	for _, container := range t.Containers {
		if capacity, ok := options.proxyOutboundCapacity[container.Image]; ok {
			sidecar.Env = append(sidecar.Env,
				v1.EnvVar{
					Name:  "LINKERD2_PROXY_OUTBOUND_ROUTER_CAPACITY",
					Value: fmt.Sprintf("%d", capacity),
				},
			)
			break
		}
	}

	if options.enableTLS() {
		yes := true

		configMapVolume := v1.Volume{
			Name: "linkerd-trust-anchors",
			VolumeSource: v1.VolumeSource{
				ConfigMap: &v1.ConfigMapVolumeSource{
					LocalObjectReference: v1.LocalObjectReference{Name: k8s.TLSTrustAnchorConfigMapName},
					Optional:             &yes,
				},
			},
		}
		secretVolume := v1.Volume{
			Name: "linkerd-secrets",
			VolumeSource: v1.VolumeSource{
				Secret: &v1.SecretVolumeSource{
					SecretName: identity.ToSecretName(),
					Optional:   &yes,
				},
			},
		}

		base := "/var/linkerd-io"
		configMapBase := base + "/trust-anchors"
		secretBase := base + "/identity"
		tlsEnvVars := []v1.EnvVar{
			{Name: "LINKERD2_PROXY_TLS_TRUST_ANCHORS", Value: configMapBase + "/" + k8s.TLSTrustAnchorFileName},
			{Name: "LINKERD2_PROXY_TLS_CERT", Value: secretBase + "/" + k8s.TLSCertFileName},
			{Name: "LINKERD2_PROXY_TLS_PRIVATE_KEY", Value: secretBase + "/" + k8s.TLSPrivateKeyFileName},
			{
				Name:  "LINKERD2_PROXY_TLS_POD_IDENTITY",
				Value: identity.ToDNSName(),
			},
			{Name: "LINKERD2_PROXY_CONTROLLER_NAMESPACE", Value: controlPlaneNamespace},
			{Name: "LINKERD2_PROXY_TLS_CONTROLLER_IDENTITY", Value: identity.ToControllerIdentity().ToDNSName()},
		}

		sidecar.Env = append(sidecar.Env, tlsEnvVars...)
		sidecar.VolumeMounts = []v1.VolumeMount{
			{Name: configMapVolume.Name, MountPath: configMapBase, ReadOnly: true},
			{Name: secretVolume.Name, MountPath: secretBase, ReadOnly: true},
		}

		t.Volumes = append(t.Volumes, configMapVolume, secretVolume)
	}

	t.Containers = append(t.Containers, sidecar)
	t.InitContainers = append(t.InitContainers, initContainer)

	return true
}

// ProcessYAML takes an input stream of YAML, outputting injected/uninjected YAML to out.
// TODO: move to separate file
func ProcessYAML(in io.Reader, out io.Writer, report io.Writer, options *injectOptions, rt resourceTransformer) error {
	reader := yamlDecoder.NewYAMLReader(bufio.NewReaderSize(in, 4096))

	injectReports := []injectReport{}

	// Iterate over all YAML objects in the input
	for {
		// Read a single YAML object
		bytes, err := reader.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			return err
		}

		result, irs, err := rt.transform(bytes, options)
		if err != nil {
			return err
		}

		out.Write(result)
		out.Write([]byte("---\n"))

		injectReports = append(injectReports, irs...)
	}

	rt.generateReport(injectReports, report)

	return nil
}

// TODO: shared with uninject, so better move it to some common file
func processList(b []byte, options *injectOptions, rt resourceTransformer) ([]byte, []injectReport, error) {
	var sourceList v1.List
	if err := yaml.Unmarshal(b, &sourceList); err != nil {
		return nil, nil, err
	}

	injectReports := []injectReport{}
	items := []runtime.RawExtension{}

	for _, item := range sourceList.Items {
		result, reports, err := rt.transform(item.Raw, options)
		if err != nil {
			return nil, nil, err
		}

		// At this point, we have yaml. The kubernetes internal representation is
		// json. Because we're building a list from RawExtensions, the yaml needs
		// to be converted to json.
		injected, err := yaml.YAMLToJSON(result)
		if err != nil {
			return nil, nil, err
		}

		items = append(items, runtime.RawExtension{Raw: injected})
		injectReports = append(injectReports, reports...)
	}

	sourceList.Items = items
	result, err := yaml.Marshal(sourceList)
	if err != nil {
		return nil, nil, err
	}

	return result, injectReports, nil
}

// TODO: move to separate file
func (conf *resourceConfig) parse(bytes []byte, options *injectOptions, rt resourceTransformer) ([]byte, []injectReport, error) {
	// The Kubernetes API is versioned and each version has an API modeled
	// with its own distinct Go types. If we tell `yaml.Unmarshal()` which
	// version we support then it will provide a representation of that
	// object using the given type if possible. However, it only allows us
	// to supply one object (of one type), so first we have to determine
	// what kind of object `bytes` represents so we can pass an object of
	// the correct type to `yaml.Unmarshal()`.
	// ---------------------------------------
	// Note: bytes is expected to be YAML and will only modify it when a
	// supported type is found. Otherwise, it is returned unmodified.

	// Unmarshal the object enough to read the Kind field
	if err := yaml.Unmarshal(bytes, &conf.meta); err != nil {
		return nil, nil, err
	}

	// retrieve the `metadata/name` field for reporting later
	if err := yaml.Unmarshal(bytes, &conf.om); err != nil {
		return nil, nil, err
	}

	conf.k8sLabels = map[string]string{}

	// When injecting the linkerd proxy into a linkerd controller pod. The linkerd proxy's
	// LINKERD2_PROXY_CONTROL_URL variable must be set to localhost for the following reasons:
	//	1. According to https://github.com/kubernetes/minikube/issues/1568, minikube has an issue
	//     where pods are unable to connect to themselves through their associated service IP.
	//     Setting the LINKERD2_PROXY_CONTROL_URL to localhost allows the proxy to bypass kube DNS
	//     name resolution as a workaround to this issue.
	//  2. We avoid the TLS overhead in encrypting and decrypting intra-pod traffic i.e. traffic
	//     between containers in the same pod.
	//  3. Using a Service IP instead of localhost would mean intra-pod traffic would be load-balanced
	//     across all controller pod replicas. This is undesirable as we would want all traffic between
	//	   containers to be self contained.
	//  4. We skip recording telemetry for intra-pod traffic within the control plane.
	switch conf.meta.Kind {
	case "Deployment":
		var deployment v1beta1.Deployment
		if err := yaml.Unmarshal(bytes, &deployment); err != nil {
			return nil, nil, err
		}

		if deployment.Name == ControlPlanePodName && deployment.Namespace == controlPlaneNamespace {
			conf.DNSNameOverride = LocalhostDNSNameOverride
		}

		conf.obj = &deployment
		conf.k8sLabels[k8s.ProxyDeploymentLabel] = deployment.Name
		conf.podSpec = &deployment.Spec.Template.Spec
		conf.objectMeta = &deployment.Spec.Template.ObjectMeta

	case "ReplicationController":
		var rc v1.ReplicationController
		if err := yaml.Unmarshal(bytes, &rc); err != nil {
			return nil, nil, err
		}

		conf.obj = &rc
		conf.k8sLabels[k8s.ProxyReplicationControllerLabel] = rc.Name
		conf.podSpec = &rc.Spec.Template.Spec
		conf.objectMeta = &rc.Spec.Template.ObjectMeta

	case "ReplicaSet":
		var rs v1beta1.ReplicaSet
		if err := yaml.Unmarshal(bytes, &rs); err != nil {
			return nil, nil, err
		}

		conf.obj = &rs
		conf.k8sLabels[k8s.ProxyReplicaSetLabel] = rs.Name
		conf.podSpec = &rs.Spec.Template.Spec
		conf.objectMeta = &rs.Spec.Template.ObjectMeta

	case "Job":
		var job batchV1.Job
		if err := yaml.Unmarshal(bytes, &job); err != nil {
			return nil, nil, err
		}

		conf.obj = &job
		conf.k8sLabels[k8s.ProxyJobLabel] = job.Name
		conf.podSpec = &job.Spec.Template.Spec
		conf.objectMeta = &job.Spec.Template.ObjectMeta

	case "DaemonSet":
		var ds v1beta1.DaemonSet
		if err := yaml.Unmarshal(bytes, &ds); err != nil {
			return nil, nil, err
		}

		conf.obj = &ds
		conf.k8sLabels[k8s.ProxyDaemonSetLabel] = ds.Name
		conf.podSpec = &ds.Spec.Template.Spec
		conf.objectMeta = &ds.Spec.Template.ObjectMeta

	case "StatefulSet":
		var statefulset appsV1.StatefulSet
		if err := yaml.Unmarshal(bytes, &statefulset); err != nil {
			return nil, nil, err
		}

		conf.obj = &statefulset
		conf.k8sLabels[k8s.ProxyStatefulSetLabel] = statefulset.Name
		conf.podSpec = &statefulset.Spec.Template.Spec
		conf.objectMeta = &statefulset.Spec.Template.ObjectMeta

	case "Pod":
		var pod v1.Pod
		if err := yaml.Unmarshal(bytes, &pod); err != nil {
			return nil, nil, err
		}

		conf.obj = &pod
		conf.podSpec = &pod.Spec
		conf.objectMeta = &pod.ObjectMeta

	case "List":
		// Lists are a little different than the other types. There's no immediate
		// pod template. Because of this, we do a recursive call for each element
		// in the list (instead of just marshaling the injected pod template).
		return processList(bytes, options, rt)
	}

	return nil, nil, nil
}

func (rt resourceTransformerInject) transform(bytes []byte, options *injectOptions) ([]byte, []injectReport, error) {
	conf := &resourceConfig{}
	output, reports, err := conf.parse(bytes, options, rt)
	if output != nil || err != nil {
		return output, reports, err
	}

	report := injectReport{
		name: fmt.Sprintf("%s/%s", strings.ToLower(conf.meta.Kind), conf.om.Name),
	}

	// If we don't inject anything into the pod template then output the
	// original serialization of the original object. Otherwise, output the
	// serialization of the modified object.
	output = bytes
	if conf.podSpec != nil {
		metaAccessor, err := k8sMeta.Accessor(conf.obj)
		if err != nil {
			return nil, nil, err
		}

		// The namespace isn't necessarily in the input so it has to be substituted
		// at runtime. The proxy recognizes the "$NAME" syntax for this variable
		// but not necessarily other variables.
		identity := k8s.TLSIdentity{
			Name:                metaAccessor.GetName(),
			Kind:                strings.ToLower(conf.meta.Kind),
			Namespace:           "$" + PodNamespaceEnvVarName,
			ControllerNamespace: controlPlaneNamespace,
		}

		if injectPodSpec(conf.podSpec, identity, conf.DNSNameOverride, options, &report) {
			injectObjectMeta(conf.objectMeta, conf.k8sLabels, options)
			var err error
			output, err = yaml.Marshal(conf.obj)
			if err != nil {
				return nil, nil, err
			}
		}
	} else {
		report.unsupportedResource = true
	}

	return output, []injectReport{report}, nil
}

func (resourceTransformerInject) generateReport(injectReports []injectReport, output io.Writer) {
	injected := []string{}
	hostNetwork := []string{}
	sidecar := []string{}
	udp := []string{}

	for _, r := range injectReports {
		if !r.hostNetwork && !r.sidecar && !r.unsupportedResource {
			injected = append(injected, r.name)
		}

		if r.hostNetwork {
			hostNetwork = append(hostNetwork, r.name)
		}

		if r.sidecar {
			sidecar = append(sidecar, r.name)
		}

		if r.udp {
			udp = append(udp, r.name)
		}
	}

	//
	// Warnings
	//

	// leading newline to separate from yaml output on stdout
	output.Write([]byte("\n"))

	hostNetworkPrefix := fmt.Sprintf("%s%s", hostNetworkDesc, getFiller(hostNetworkDesc))
	if len(hostNetwork) == 0 {
		output.Write([]byte(fmt.Sprintf("%s%s\n", hostNetworkPrefix, okStatus)))
	} else {
		output.Write([]byte(fmt.Sprintf("%s%s -- \"hostNetwork: true\" detected in %s\n", hostNetworkPrefix, warnStatus, strings.Join(hostNetwork, ", "))))
	}

	sidecarPrefix := fmt.Sprintf("%s%s", sidecarDesc, getFiller(sidecarDesc))
	if len(sidecar) == 0 {
		output.Write([]byte(fmt.Sprintf("%s%s\n", sidecarPrefix, okStatus)))
	} else {
		output.Write([]byte(fmt.Sprintf("%s%s -- known sidecar detected in %s\n", sidecarPrefix, warnStatus, strings.Join(sidecar, ", "))))
	}

	unsupportedPrefix := fmt.Sprintf("%s%s", unsupportedDesc, getFiller(unsupportedDesc))
	if len(injected) > 0 {
		output.Write([]byte(fmt.Sprintf("%s%s\n", unsupportedPrefix, okStatus)))
	} else {
		output.Write([]byte(fmt.Sprintf("%s%s -- no supported objects found\n", unsupportedPrefix, warnStatus)))
	}

	udpPrefix := fmt.Sprintf("%s%s", udpDesc, getFiller(udpDesc))
	if len(udp) == 0 {
		output.Write([]byte(fmt.Sprintf("%s%s\n", udpPrefix, okStatus)))
	} else {
		verb := "uses"
		if len(udp) > 1 {
			verb = "use"
		}
		output.Write([]byte(fmt.Sprintf("%s%s -- %s %s \"protocol: UDP\"\n", udpPrefix, warnStatus, strings.Join(udp, ", "), verb)))
	}

	//
	// Summary
	//

	summary := fmt.Sprintf("Summary: %d of %d YAML document(s) injected", len(injected), len(injectReports))
	output.Write([]byte(fmt.Sprintf("\n%s\n", summary)))

	for _, i := range injected {
		output.Write([]byte(fmt.Sprintf("  %s\n", i)))
	}

	// trailing newline to separate from kubectl output if piping
	output.Write([]byte("\n"))
}

func checkUDPPorts(t *v1.PodSpec) bool {
	// check for ports with `protocol: UDP`, which will not be routed by Linkerd
	for _, container := range t.Containers {
		for _, port := range container.Ports {
			if port.Protocol == v1.ProtocolUDP {
				return true
			}
		}
	}
	return false
}
