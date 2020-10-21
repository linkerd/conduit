package inject

import (
	"reflect"
	"testing"

	l5dcharts "github.com/linkerd/linkerd2/pkg/charts/linkerd2"
	"github.com/linkerd/linkerd2/pkg/k8s"
	"github.com/linkerd/linkerd2/pkg/version"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/yaml"
)

func TestGetOverriddenValues(t *testing.T) {
	// this test uses an annotated deployment and a expected Values object to verify
	// the GetOverriddenValues function.

	var (
		proxyVersionOverride = "proxy-version-override"
		pullPolicy           = "Always"
	)

	testConfig, err := l5dcharts.NewValues(false)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	var testCases = []struct {
		id            string
		nsAnnotations map[string]string
		spec          appsv1.DeploymentSpec
		expected      func() *l5dcharts.Values
	}{
		{id: "use overrides",
			nsAnnotations: make(map[string]string),
			spec: appsv1.DeploymentSpec{
				Template: corev1.PodTemplateSpec{
					ObjectMeta: metav1.ObjectMeta{
						Annotations: map[string]string{
							k8s.ProxyDisableIdentityAnnotation:               "true",
							k8s.ProxyImageAnnotation:                         "ghcr.io/linkerd/proxy",
							k8s.ProxyImagePullPolicyAnnotation:               pullPolicy,
							k8s.ProxyInitImageAnnotation:                     "ghcr.io/linkerd/proxy-init",
							k8s.ProxyControlPortAnnotation:                   "4000",
							k8s.ProxyInboundPortAnnotation:                   "5000",
							k8s.ProxyAdminPortAnnotation:                     "5001",
							k8s.ProxyOutboundPortAnnotation:                  "5002",
							k8s.ProxyIgnoreInboundPortsAnnotation:            "4222,6222",
							k8s.ProxyIgnoreOutboundPortsAnnotation:           "8079,8080",
							k8s.ProxyCPURequestAnnotation:                    "0.15",
							k8s.ProxyMemoryRequestAnnotation:                 "120",
							k8s.ProxyCPULimitAnnotation:                      "1.5",
							k8s.ProxyMemoryLimitAnnotation:                   "256",
							k8s.ProxyUIDAnnotation:                           "8500",
							k8s.ProxyLogLevelAnnotation:                      "debug,linkerd=debug",
							k8s.ProxyLogFormatAnnotation:                     "json",
							k8s.ProxyEnableExternalProfilesAnnotation:        "false",
							k8s.ProxyVersionOverrideAnnotation:               proxyVersionOverride,
							k8s.ProxyTraceCollectorSvcAddrAnnotation:         "oc-collector.tracing:55678",
							k8s.ProxyTraceCollectorSvcAccountAnnotation:      "default",
							k8s.ProxyWaitBeforeExitSecondsAnnotation:         "123",
							k8s.ProxyRequireIdentityOnInboundPortsAnnotation: "8888,9999",
							k8s.ProxyDestinationGetNetworks:                  "10.0.0.0/8",
							k8s.ProxyOutboundConnectTimeout:                  "6000ms",
							k8s.ProxyInboundConnectTimeout:                   "600ms",
							k8s.ProxyOpaquePortsAnnotation:                   "4320-4325,3306",
						},
					},
					Spec: corev1.PodSpec{},
				},
			},
			expected: func() *l5dcharts.Values {
				values, _ := l5dcharts.NewValues(false)

				values.Global.Proxy.DisableIdentity = true
				values.Global.ClusterNetworks = "10.0.0.0/8"
				values.Global.Proxy.Image.Name = "ghcr.io/linkerd/proxy"
				values.Global.Proxy.Image.PullPolicy = pullPolicy
				values.Global.Proxy.Image.Version = proxyVersionOverride
				values.Global.Proxy.Ports.Control = 4000
				values.Global.Proxy.Ports.Inbound = 5000
				values.Global.Proxy.Ports.Admin = 5001
				values.Global.Proxy.Ports.Outbound = 5002
				values.Global.Proxy.WaitBeforeExitSeconds = 123
				values.Global.Proxy.LogLevel = "debug,linkerd=debug"
				values.Global.Proxy.LogFormat = "json"
				values.Global.Proxy.Resources = &l5dcharts.Resources{
					CPU: l5dcharts.Constraints{
						Limit:   "1.5",
						Request: "0.15",
					},
					Memory: l5dcharts.Constraints{
						Limit:   "256",
						Request: "120",
					},
				}
				values.Global.Proxy.UID = 8500
				values.Global.ProxyInit.Image.Name = "ghcr.io/linkerd/proxy-init"
				values.Global.ProxyInit.Image.PullPolicy = pullPolicy
				values.Global.ProxyInit.Image.Version = version.ProxyInitVersion
				values.Global.ProxyInit.IgnoreInboundPorts = "4222,6222"
				values.Global.ProxyInit.IgnoreOutboundPorts = "8079,8080"
				values.Global.Proxy.Trace = &l5dcharts.Trace{
					CollectorSvcAddr:    "oc-collector.tracing:55678",
					CollectorSvcAccount: "default.tracing",
				}
				values.Global.Proxy.RequireIdentityOnInboundPorts = "8888,9999"
				values.Global.Proxy.OutboundConnectTimeout = "6000ms"
				values.Global.Proxy.InboundConnectTimeout = "600ms"
				values.Global.Proxy.OpaquePorts = "4320,4321,4322,4323,4324,4325,3306"
				return values
			},
		},
		{id: "use defaults",
			nsAnnotations: make(map[string]string),
			spec: appsv1.DeploymentSpec{
				Template: corev1.PodTemplateSpec{
					ObjectMeta: metav1.ObjectMeta{},
					Spec:       corev1.PodSpec{},
				},
			},
			expected: func() *l5dcharts.Values {
				values, _ := l5dcharts.NewValues(false)
				return values
			},
		},
		{id: "use namespace overrides",
			nsAnnotations: map[string]string{
				k8s.ProxyDisableIdentityAnnotation:          "true",
				k8s.ProxyImageAnnotation:                    "ghcr.io/linkerd/proxy",
				k8s.ProxyImagePullPolicyAnnotation:          pullPolicy,
				k8s.ProxyInitImageAnnotation:                "ghcr.io/linkerd/proxy-init",
				k8s.ProxyControlPortAnnotation:              "4000",
				k8s.ProxyInboundPortAnnotation:              "5000",
				k8s.ProxyAdminPortAnnotation:                "5001",
				k8s.ProxyOutboundPortAnnotation:             "5002",
				k8s.ProxyIgnoreInboundPortsAnnotation:       "4222,6222",
				k8s.ProxyIgnoreOutboundPortsAnnotation:      "8079,8080",
				k8s.ProxyCPURequestAnnotation:               "0.15",
				k8s.ProxyMemoryRequestAnnotation:            "120",
				k8s.ProxyCPULimitAnnotation:                 "1.5",
				k8s.ProxyMemoryLimitAnnotation:              "256",
				k8s.ProxyUIDAnnotation:                      "8500",
				k8s.ProxyLogLevelAnnotation:                 "debug,linkerd=debug",
				k8s.ProxyLogFormatAnnotation:                "json",
				k8s.ProxyEnableExternalProfilesAnnotation:   "false",
				k8s.ProxyVersionOverrideAnnotation:          proxyVersionOverride,
				k8s.ProxyTraceCollectorSvcAddrAnnotation:    "oc-collector.tracing:55678",
				k8s.ProxyTraceCollectorSvcAccountAnnotation: "default",
				k8s.ProxyWaitBeforeExitSecondsAnnotation:    "123",
				k8s.ProxyOutboundConnectTimeout:             "6000ms",
				k8s.ProxyInboundConnectTimeout:              "600ms",
				k8s.ProxyOpaquePortsAnnotation:              "4320-4325,3306",
			},
			spec: appsv1.DeploymentSpec{
				Template: corev1.PodTemplateSpec{
					Spec: corev1.PodSpec{},
				},
			},
			expected: func() *l5dcharts.Values {
				values, _ := l5dcharts.NewValues(false)

				values.Global.Proxy.DisableIdentity = true
				values.Global.Proxy.Image.Name = "ghcr.io/linkerd/proxy"
				values.Global.Proxy.Image.PullPolicy = pullPolicy
				values.Global.Proxy.Image.Version = proxyVersionOverride
				values.Global.Proxy.Ports.Control = 4000
				values.Global.Proxy.Ports.Inbound = 5000
				values.Global.Proxy.Ports.Admin = 5001
				values.Global.Proxy.Ports.Outbound = 5002
				values.Global.Proxy.WaitBeforeExitSeconds = 123
				values.Global.Proxy.LogLevel = "debug,linkerd=debug"
				values.Global.Proxy.LogFormat = "json"
				values.Global.Proxy.Resources = &l5dcharts.Resources{
					CPU: l5dcharts.Constraints{
						Limit:   "1.5",
						Request: "0.15",
					},
					Memory: l5dcharts.Constraints{
						Limit:   "256",
						Request: "120",
					},
				}
				values.Global.Proxy.UID = 8500
				values.Global.ProxyInit.Image.Name = "ghcr.io/linkerd/proxy-init"
				values.Global.ProxyInit.Image.PullPolicy = pullPolicy
				values.Global.ProxyInit.Image.Version = version.ProxyInitVersion
				values.Global.ProxyInit.IgnoreInboundPorts = "4222,6222"
				values.Global.ProxyInit.IgnoreOutboundPorts = "8079,8080"
				values.Global.Proxy.Trace = &l5dcharts.Trace{
					CollectorSvcAddr:    "oc-collector.tracing:55678",
					CollectorSvcAccount: "default.tracing",
				}
				values.Global.Proxy.OutboundConnectTimeout = "6000ms"
				values.Global.Proxy.InboundConnectTimeout = "600ms"
				values.Global.Proxy.OpaquePorts = "4320,4321,4322,4323,4324,4325,3306"
				return values
			},
		},
		{id: "use empty string for dst networks",
			nsAnnotations: map[string]string{
				k8s.ProxyDestinationGetNetworks: "",
			},
			spec: appsv1.DeploymentSpec{
				Template: corev1.PodTemplateSpec{
					ObjectMeta: metav1.ObjectMeta{},
					Spec:       corev1.PodSpec{},
				},
			},
			expected: func() *l5dcharts.Values {
				values, _ := l5dcharts.NewValues(false)
				values.Global.ClusterNetworks = ""
				return values
			},
		},
		{id: "use invalid duration for TCP connect timeouts",
			nsAnnotations: map[string]string{
				k8s.ProxyOutboundConnectTimeout: "6000",
				k8s.ProxyInboundConnectTimeout:  "600",
			},
			spec: appsv1.DeploymentSpec{
				Template: corev1.PodTemplateSpec{
					ObjectMeta: metav1.ObjectMeta{},
					Spec:       corev1.PodSpec{},
				},
			},
			expected: func() *l5dcharts.Values {
				values, _ := l5dcharts.NewValues(false)
				return values
			},
		},
		{id: "use valid duration for TCP connect timeouts",
			nsAnnotations: map[string]string{
				// Validate we're converting time values into ms for the proxy to parse correctly.
				k8s.ProxyOutboundConnectTimeout: "6s5ms",
				k8s.ProxyInboundConnectTimeout:  "2s5ms",
			},
			spec: appsv1.DeploymentSpec{
				Template: corev1.PodTemplateSpec{
					ObjectMeta: metav1.ObjectMeta{},
					Spec:       corev1.PodSpec{},
				},
			},
			expected: func() *l5dcharts.Values {
				values, _ := l5dcharts.NewValues(false)
				values.Global.Proxy.OutboundConnectTimeout = "6005ms"
				values.Global.Proxy.InboundConnectTimeout = "2005ms"
				return values
			},
		},
		{id: "use named port for opaque ports",
			nsAnnotations: make(map[string]string),
			spec: appsv1.DeploymentSpec{
				Template: corev1.PodTemplateSpec{
					ObjectMeta: metav1.ObjectMeta{
						Annotations: map[string]string{
							k8s.ProxyOpaquePortsAnnotation: "mysql",
						},
					},
					Spec: corev1.PodSpec{
						Containers: []corev1.Container{
							corev1.Container{
								Ports: []corev1.ContainerPort{
									corev1.ContainerPort{
										Name:          "mysql",
										ContainerPort: 3306,
									},
								},
							},
						},
					},
				},
			},
			expected: func() *l5dcharts.Values {
				values, _ := l5dcharts.NewValues(false)
				values.Global.Proxy.OpaquePorts = "3306"
				return values
			},
		},
	}

	for _, tc := range testCases {
		testCase := tc
		t.Run(testCase.id, func(t *testing.T) {
			data, err := yaml.Marshal(&appsv1.Deployment{Spec: testCase.spec})
			if err != nil {
				t.Fatal(err)
			}

			resourceConfig := NewResourceConfig(testConfig, OriginUnknown).WithKind("Deployment").WithNsAnnotations(testCase.nsAnnotations)
			if err := resourceConfig.parse(data); err != nil {
				t.Fatal(err)
			}

			actual, err := resourceConfig.GetOverriddenValues()
			if err != nil {
				t.Fatal(err)
			}
			expected := testCase.expected()
			if !reflect.DeepEqual(actual, expected) {
				t.Fatalf("Expected values to be \n%v\n but was \n%v", expected.String(), actual.String())
			}

		})
	}
}
