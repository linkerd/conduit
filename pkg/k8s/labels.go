/*
Kubernetes labels and annotations used in Linkerd's control plane and data plane
Kubernetes configs.
*/

package k8s

import (
	"fmt"

	"github.com/linkerd/linkerd2/pkg/version"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
)

const (
	/*
	 * Labels
	 */

	// Prefix is the prefix common to all labels and annotations injected by Linkerd
	Prefix = "linkerd.io"

	// ControllerComponentLabel identifies this object as a component of Linkerd's
	// control plane (e.g. web, controller).
	ControllerComponentLabel = Prefix + "/control-plane-component"

	// ControllerNSLabel is injected into mesh-enabled apps, identifying the
	// namespace of the Linkerd control plane.
	ControllerNSLabel = Prefix + "/control-plane-ns"

	// ProxyDeploymentLabel is injected into mesh-enabled apps, identifying the
	// deployment that this proxy belongs to.
	ProxyDeploymentLabel = Prefix + "/proxy-deployment"

	// ProxyReplicationControllerLabel is injected into mesh-enabled apps,
	// identifying the ReplicationController that this proxy belongs to.
	ProxyReplicationControllerLabel = Prefix + "/proxy-replicationcontroller"

	// ProxyReplicaSetLabel is injected into mesh-enabled apps, identifying the
	// ReplicaSet that this proxy belongs to.
	ProxyReplicaSetLabel = Prefix + "/proxy-replicaset"

	// ProxyJobLabel is injected into mesh-enabled apps, identifying the Job that
	// this proxy belongs to.
	ProxyJobLabel = Prefix + "/proxy-job"

	// ProxyDaemonSetLabel is injected into mesh-enabled apps, identifying the
	// DaemonSet that this proxy belongs to.
	ProxyDaemonSetLabel = Prefix + "/proxy-daemonset"

	// ProxyStatefulSetLabel is injected into mesh-enabled apps, identifying the
	// StatefulSet that this proxy belongs to.
	ProxyStatefulSetLabel = Prefix + "/proxy-statefulset"

	/*
	 * Annotations
	 */

	// CreatedByAnnotation indicates the source of the injected data plane
	// (e.g. linkerd/cli v2.0.0).
	CreatedByAnnotation = Prefix + "/created-by"

	// ProxyVersionAnnotation indicates the version of the injected data plane
	// (e.g. v0.1.3).
	ProxyVersionAnnotation = Prefix + "/proxy-version"

	// ProxyInjectAnnotation controls whether or not a pod should be injected
	// when set on a pod spec. When set on a namespace spec, it applies to all
	// pods in the namespace. Supported values are "enabled" or "disabled"
	ProxyInjectAnnotation = Prefix + "/inject"

	// ProxyInjectEnabled is assigned to the ProxyInjectAnnotation annotation to
	// enable injection for a pod or namespace.
	ProxyInjectEnabled = "enabled"

	// ProxyInjectDisabled is assigned to the ProxyInjectAnnotation annotation to
	// disable injection for a pod or namespace.
	ProxyInjectDisabled = "disabled"

	// IdentityModeAnnotation controls how a pod participates
	// in service identity.
	IdentityModeAnnotation = Prefix + "/identity-mode"

	// IdentityModeDisabled is assigned to IdentityModeAnnotation to
	// disable the proxy from participating in automatic identity.
	IdentityModeDisabled = "disabled"

	// IdentityModeOptional is assigned to IdentityModeAnnotation to
	// optionally configure the proxy to participate in automatic identity.
	//
	// Will be deprecated soon.
	IdentityModeOptional = "optional"

	/*
	 * Component Names
	 */

	// InitContainerName is the name assigned to the injected init container.
	InitContainerName = "linkerd-init"

	// ProxyContainerName is the name assigned to the injected proxy container.
	ProxyContainerName = "linkerd-proxy"

	// ProxyPortName is the name of the Linkerd Proxy's proxy port
	ProxyPortName = "linkerd-proxy"

	// ProxyMetricsPortName is the name of the Linkerd Proxy's metrics port
	ProxyMetricsPortName = "linkerd-metrics"

	// ProxyInjectorWebhookConfig is the name of the mutating webhook
	// configuration resource of the proxy-injector webhook.
	ProxyInjectorWebhookConfig = "linkerd-proxy-injector-webhook-config"

	// TLSTrustAnchorVolumeName is the name of the trust anchor volume,
	// used when injecting a proxy with TLS enabled.
	TLSTrustAnchorVolumeName = "linkerd-trust-anchors"

	// TLSSecretsVolumeName is the name of the volume holding the secrets,
	// when injecting a proxy with TLS enabled.
	TLSSecretsVolumeName = "linkerd-secrets"

	// TLSTrustAnchorConfigMapName is the name of the ConfigMap that holds the
	// trust anchors (trusted root certificates).
	TLSTrustAnchorConfigMapName = "linkerd-ca-bundle"

	// TLSTrustAnchorFileName is the name (key) within the trust anchor ConfigMap
	// that contains the actual trust anchor bundle.
	TLSTrustAnchorFileName = "trust-anchors.pem"

	// TLSCertFileName is the name (key) within proxy-injector ConfigMap that
	// contains the TLS certificate.
	TLSCertFileName = "certificate.crt"

	// TLSPrivateKeyFileName is the name (key) within proxy-injector ConfigMap
	// that contains the TLS private key.
	TLSPrivateKeyFileName = "private-key.p8"

	/*
	 * Mount paths
	 */

	// MountPathBase is the base directory of the mount path
	MountPathBase = "/var/linkerd-io"
)

var (
	// MountPathTLSTrustAnchor is the path at which the trust anchor file is
	// mounted
	MountPathTLSTrustAnchor = MountPathBase + "/trust-anchors/" + TLSTrustAnchorFileName

	// MountPathTLSIdentityCert is the path at which the TLS identity cert file is
	// mounted
	MountPathTLSIdentityCert = MountPathBase + "/identity/" + TLSCertFileName

	// MountPathTLSIdentityKey is the path at which the TLS identity key file is
	// mounted
	MountPathTLSIdentityKey = MountPathBase + "/identity/" + TLSPrivateKeyFileName

	// MountPathGlobalConfig is the path at which the global config file is mounted
	MountPathGlobalConfig = MountPathBase + "/config/global"

	// MountPathProxyConfig is the path at which the global config file is mounted
	MountPathProxyConfig = MountPathBase + "/config/proxy"
)

// CreatedByAnnotationValue returns the value associated with
// CreatedByAnnotation.
func CreatedByAnnotationValue() string {
	return fmt.Sprintf("linkerd/cli %s", version.Version)
}

// GetPodLabels returns the set of prometheus owner labels for a given pod
func GetPodLabels(ownerKind, ownerName string, pod *corev1.Pod) map[string]string {
	labels := map[string]string{"pod": pod.Name}

	l5dLabel := KindToL5DLabel(ownerKind)
	labels[l5dLabel] = ownerName

	if controllerNS := pod.Labels[ControllerNSLabel]; controllerNS != "" {
		labels["control_plane_ns"] = controllerNS
	}

	if pth := pod.Labels[appsv1.DefaultDeploymentUniqueLabelKey]; pth != "" {
		labels["pod_template_hash"] = pth
	}

	return labels
}

// IsMeshed returns whether a given Pod is in a given controller's service mesh.
func IsMeshed(pod *corev1.Pod, controllerNS string) bool {
	return pod.Labels[ControllerNSLabel] == controllerNS
}

// TLSIdentity is the identity of a pod owner (Deployment, Pod,
// ReplicationController, etc.).
type TLSIdentity struct {
	// Name is the name of the pod owner.
	Name string

	// Kind is the singular, lowercased Kubernetes resource type of the pod owner
	// (deployment, daemonset, job, replicationcontroller, etc.).
	Kind string

	// Namespace is the pod's namespace. Kubernetes requires that pods and
	// pod owners be in the same namespace.
	Namespace string

	// ControllerNamespace is the namespace of the controller for the pod.
	ControllerNamespace string
}

// ToDNSName formats a TLSIdentity as a DNS name.
func (i TLSIdentity) ToDNSName() string {
	if i.Kind == Service {
		return fmt.Sprintf("%s.%s.svc", i.Name, i.Namespace)
	}
	return fmt.Sprintf("%s.%s.%s.linkerd-managed.%s.svc.cluster.local", i.Name,
		i.Kind, i.Namespace, i.ControllerNamespace)
}

// ToSecretName formats a TLSIdentity as a secret name.
func (i TLSIdentity) ToSecretName() string {
	return fmt.Sprintf("%s-%s-tls-linkerd-io", i.Name, i.Kind)
}

// ToControllerIdentity returns the TLSIdentity of the Linkerd Controller, given
// an arbitrary TLSIdentity.
func (i TLSIdentity) ToControllerIdentity() TLSIdentity {
	return TLSIdentity{
		Name:                "linkerd-controller",
		Kind:                "deployment",
		Namespace:           i.ControllerNamespace,
		ControllerNamespace: i.ControllerNamespace,
	}
}
