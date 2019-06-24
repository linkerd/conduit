package inject

import (
	"fmt"
	"reflect"
	"testing"

	"github.com/linkerd/linkerd2/controller/gen/config"
	"github.com/linkerd/linkerd2/pkg/k8s"
	"github.com/linkerd/linkerd2/pkg/version"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	k8sResource "k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	"sigs.k8s.io/yaml"
)

type expectedProxyConfigs struct {
	identityContext            *config.IdentityContext
	image                      string
	imagePullPolicy            corev1.PullPolicy
	proxyVersion               string
	controlPort                int32
	inboundPort                int32
	adminPort                  int32
	outboundPort               int32
	logLevel                   string
	resourceRequirements       corev1.ResourceRequirements
	destinationAddr            string
	controlListenAddr          string
	inboundListenAddr          string
	metricsListenAddr          string
	outboundListenAddr         string
	proxyUID                   int64
	livenessProbe              *corev1.Probe
	readinessProbe             *corev1.Probe
	destinationProfileSuffixes string
	initImage                  string
	initImagePullPolicy        corev1.PullPolicy
	initVersion                string
	initArgs                   []string
	inboundSkipPorts           string
	outboundSkipPorts          string
}

func TestConfigAccessors(t *testing.T) {
	// this test uses an annotated deployment and a proxyConfig object to verify
	// all the proxy config accessors. The first test suite ensures that the
	// accessors picks up the pod-level config annotations. The second test suite
	// ensures that the defaults in the config map is used.

	var (
		controlPlaneVersion  = "control-plane-version"
		proxyVersion         = "proxy-version"
		proxyVersionOverride = "proxy-version-override"
	)

	proxyConfig := &config.Proxy{
		ProxyImage:          &config.Image{ImageName: "gcr.io/linkerd-io/proxy", PullPolicy: "IfNotPresent"},
		ProxyInitImage:      &config.Image{ImageName: "gcr.io/linkerd-io/proxy-init", PullPolicy: "IfNotPresent"},
		ControlPort:         &config.Port{Port: 9000},
		InboundPort:         &config.Port{Port: 6000},
		AdminPort:           &config.Port{Port: 6001},
		OutboundPort:        &config.Port{Port: 6002},
		IgnoreInboundPorts:  []*config.Port{{Port: 53}},
		IgnoreOutboundPorts: []*config.Port{{Port: 9079}},
		Resource: &config.ResourceRequirements{
			RequestCpu:    "0.2",
			RequestMemory: "64",
			LimitCpu:      "1",
			LimitMemory:   "128",
		},
		ProxyUid:                8888,
		LogLevel:                &config.LogLevel{Level: "info,linkerd2_proxy=debug"},
		DisableExternalProfiles: false,
		ProxyVersion:            proxyVersion,
	}

	globalConfig := &config.Global{
		LinkerdNamespace: "linkerd",
		Version:          controlPlaneVersion,
		IdentityContext:  &config.IdentityContext{},
		ClusterDomain:    "cluster.local",
	}

	configs := &config.All{Global: globalConfig, Proxy: proxyConfig}

	var testCases = []struct {
		id       string
		spec     appsv1.DeploymentSpec
		expected expectedProxyConfigs
	}{
		{id: "use overrides",
			spec: appsv1.DeploymentSpec{
				Template: corev1.PodTemplateSpec{
					metav1.ObjectMeta{
						Annotations: map[string]string{
							k8s.ProxyDisableIdentityAnnotation:        "true",
							k8s.ProxyImageAnnotation:                  "gcr.io/linkerd-io/proxy",
							k8s.ProxyImagePullPolicyAnnotation:        "Always",
							k8s.ProxyInitImageAnnotation:              "gcr.io/linkerd-io/proxy-init",
							k8s.ProxyControlPortAnnotation:            "4000",
							k8s.ProxyInboundPortAnnotation:            "5000",
							k8s.ProxyAdminPortAnnotation:              "5001",
							k8s.ProxyOutboundPortAnnotation:           "5002",
							k8s.ProxyIgnoreInboundPortsAnnotation:     "4222,6222",
							k8s.ProxyIgnoreOutboundPortsAnnotation:    "8079,8080",
							k8s.ProxyCPURequestAnnotation:             "0.15",
							k8s.ProxyMemoryRequestAnnotation:          "120",
							k8s.ProxyCPULimitAnnotation:               "1.5",
							k8s.ProxyMemoryLimitAnnotation:            "256",
							k8s.ProxyUIDAnnotation:                    "8500",
							k8s.ProxyLogLevelAnnotation:               "debug,linkerd2_proxy=debug",
							k8s.ProxyEnableExternalProfilesAnnotation: "false",
							k8s.ProxyVersionOverrideAnnotation:        proxyVersionOverride},
					},
					corev1.PodSpec{},
				},
			},
			expected: expectedProxyConfigs{
				image:           "gcr.io/linkerd-io/proxy",
				imagePullPolicy: corev1.PullPolicy("Always"),
				proxyVersion:    proxyVersionOverride,
				controlPort:     int32(4000),
				inboundPort:     int32(5000),
				adminPort:       int32(5001),
				outboundPort:    int32(5002),
				logLevel:        "debug,linkerd2_proxy=debug",
				resourceRequirements: corev1.ResourceRequirements{
					Requests: corev1.ResourceList{
						"cpu":    k8sResource.MustParse("0.15"),
						"memory": k8sResource.MustParse("120"),
					},
					Limits: corev1.ResourceList{
						"cpu":    k8sResource.MustParse("1.5"),
						"memory": k8sResource.MustParse("256"),
					},
				},
				destinationAddr:    "linkerd-destination.linkerd.svc.cluster.local:8086",
				controlListenAddr:  "0.0.0.0:4000",
				inboundListenAddr:  "0.0.0.0:5000",
				metricsListenAddr:  "0.0.0.0:5001",
				outboundListenAddr: "127.0.0.1:5002",
				proxyUID:           int64(8500),
				livenessProbe: &corev1.Probe{
					Handler: corev1.Handler{
						HTTPGet: &corev1.HTTPGetAction{
							Path: "/metrics",
							Port: intstr.IntOrString{IntVal: int32(5001)},
						},
					},
					InitialDelaySeconds: 10,
				},
				readinessProbe: &corev1.Probe{
					Handler: corev1.Handler{
						HTTPGet: &corev1.HTTPGetAction{
							Path: "/ready",
							Port: intstr.IntOrString{IntVal: int32(5001)},
						},
					},
					InitialDelaySeconds: 2,
				},
				destinationProfileSuffixes: "svc.cluster.local.",
				initImage:                  "gcr.io/linkerd-io/proxy-init",
				initImagePullPolicy:        corev1.PullPolicy("Always"),
				initVersion:                version.ProxyInitVersion,
				initArgs: []string{
					"--incoming-proxy-port", "5000",
					"--outgoing-proxy-port", "5002",
					"--proxy-uid", "8500",
					"--inbound-ports-to-ignore", "4222,6222,4000,5001",
					"--outbound-ports-to-ignore", "8079,8080",
				},
				inboundSkipPorts:  "4222,6222",
				outboundSkipPorts: "8079,8080",
			},
		},
		{id: "use defaults",
			spec: appsv1.DeploymentSpec{
				Template: corev1.PodTemplateSpec{
					metav1.ObjectMeta{},
					corev1.PodSpec{},
				},
			},
			expected: expectedProxyConfigs{
				identityContext: &config.IdentityContext{},
				image:           "gcr.io/linkerd-io/proxy",
				imagePullPolicy: corev1.PullPolicy("IfNotPresent"),
				proxyVersion:    proxyVersion,
				controlPort:     int32(9000),
				inboundPort:     int32(6000),
				adminPort:       int32(6001),
				outboundPort:    int32(6002),
				logLevel:        "info,linkerd2_proxy=debug",
				resourceRequirements: corev1.ResourceRequirements{
					Requests: corev1.ResourceList{
						"cpu":    k8sResource.MustParse("0.2"),
						"memory": k8sResource.MustParse("64"),
					},
					Limits: corev1.ResourceList{
						"cpu":    k8sResource.MustParse("1"),
						"memory": k8sResource.MustParse("128"),
					},
				},
				destinationAddr:    "linkerd-destination.linkerd.svc.cluster.local:8086",
				controlListenAddr:  "0.0.0.0:9000",
				inboundListenAddr:  "0.0.0.0:6000",
				metricsListenAddr:  "0.0.0.0:6001",
				outboundListenAddr: "127.0.0.1:6002",
				proxyUID:           int64(8888),
				livenessProbe: &corev1.Probe{
					Handler: corev1.Handler{
						HTTPGet: &corev1.HTTPGetAction{
							Path: "/metrics",
							Port: intstr.IntOrString{IntVal: int32(6001)},
						},
					},
					InitialDelaySeconds: 10,
				},
				readinessProbe: &corev1.Probe{
					Handler: corev1.Handler{
						HTTPGet: &corev1.HTTPGetAction{
							Path: "/ready",
							Port: intstr.IntOrString{IntVal: int32(6001)},
						},
					},
					InitialDelaySeconds: 2,
				},
				destinationProfileSuffixes: ".",
				initImage:                  "gcr.io/linkerd-io/proxy-init",
				initImagePullPolicy:        corev1.PullPolicy("IfNotPresent"),
				initVersion:                version.ProxyInitVersion,
				initArgs: []string{
					"--incoming-proxy-port", "6000",
					"--outgoing-proxy-port", "6002",
					"--proxy-uid", "8888",
					"--inbound-ports-to-ignore", "53,9000,6001",
					"--outbound-ports-to-ignore", "9079",
				},
				inboundSkipPorts:  "53",
				outboundSkipPorts: "9079",
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

			resourceConfig := NewResourceConfig(configs, OriginUnknown).WithKind("Deployment")
			if err := resourceConfig.parse(data); err != nil {
				t.Fatal(err)
			}

			t.Run("identityContext", func(t *testing.T) {
				expected := testCase.expected.identityContext
				if actual := resourceConfig.identityContext(); !reflect.DeepEqual(expected, actual) {
					t.Errorf("Expected: %+v Actual: %+v", expected, actual)
				}
			})

			t.Run("proxyImage", func(t *testing.T) {
				expected := testCase.expected.image
				if actual := resourceConfig.proxyImage(); expected != actual {
					t.Errorf("Expected: %v Actual: %v", expected, actual)
				}
			})

			t.Run("proxyImagePullPolicy", func(t *testing.T) {
				expected := testCase.expected.imagePullPolicy
				if actual := resourceConfig.proxyImagePullPolicy(); expected != actual {
					t.Errorf("Expected: %v Actual: %v", expected, actual)
				}
			})

			t.Run("proxyVersion", func(t *testing.T) {
				expected := testCase.expected.proxyVersion
				if actual := resourceConfig.proxyVersion(); expected != actual {
					t.Errorf("Expected: %v Actual: %v", expected, actual)
				}
			})

			t.Run("proxyInitVersion", func(t *testing.T) {
				expected := testCase.expected.initVersion
				if actual := resourceConfig.proxyInitVersion(); expected != actual {
					t.Errorf("Expected: %v Actual: %v", expected, actual)
				}
			})

			t.Run("proxyControlPort", func(t *testing.T) {
				expected := testCase.expected.controlPort
				if actual := resourceConfig.proxyControlPort(); expected != actual {
					t.Errorf("Expected: %v Actual: %v", expected, actual)
				}
			})

			t.Run("proxyInboundPort", func(t *testing.T) {
				expected := testCase.expected.inboundPort
				if actual := resourceConfig.proxyInboundPort(); expected != actual {
					t.Errorf("Expected: %v Actual: %v", expected, actual)
				}
			})

			t.Run("proxyAdminPort", func(t *testing.T) {
				expected := testCase.expected.adminPort
				if actual := resourceConfig.proxyAdminPort(); expected != actual {
					t.Errorf("Expected: %v Actual: %v", expected, actual)
				}
			})

			t.Run("proxyOutboundPort", func(t *testing.T) {
				expected := testCase.expected.outboundPort
				if actual := resourceConfig.proxyOutboundPort(); expected != actual {
					t.Errorf("Expected: %v Actual: %v", expected, actual)
				}
			})

			t.Run("proxyLogLevel", func(t *testing.T) {
				expected := testCase.expected.logLevel
				if actual := resourceConfig.proxyLogLevel(); expected != actual {
					t.Errorf("Expected: %v Actual: %v", expected, actual)
				}
			})

			t.Run("proxyResourceRequirements", func(t *testing.T) {
				expected := testCase.expected.resourceRequirements
				if actual := resourceConfig.proxyResourceRequirements(); !reflect.DeepEqual(expected, actual) {
					t.Errorf("Expected: %v Actual: %v", expected, actual)
				}
			})

			t.Run("proxyControlURL", func(t *testing.T) {
				expected := testCase.expected.destinationAddr
				if actual := resourceConfig.proxyDestinationAddr(); expected != actual {
					t.Errorf("Expected: %v Actual: %v", expected, actual)
				}
			})

			t.Run("proxyControlListenAddr", func(t *testing.T) {
				expected := testCase.expected.controlListenAddr
				if actual := resourceConfig.proxyControlListenAddr(); expected != actual {
					t.Errorf("Expected: %v Actual: %v", expected, actual)
				}
			})

			t.Run("proxyInboundListenAddr", func(t *testing.T) {
				expected := testCase.expected.inboundListenAddr
				if actual := resourceConfig.proxyInboundListenAddr(); expected != actual {
					t.Errorf("Expected: %v Actual: %v", expected, actual)
				}
			})

			t.Run("proxyAdminListenAddr", func(t *testing.T) {
				expected := testCase.expected.metricsListenAddr
				if actual := resourceConfig.proxyAdminListenAddr(); expected != actual {
					t.Errorf("Expected: %v Actual: %v", expected, actual)
				}
			})

			t.Run("proxyOutboundListenAddr", func(t *testing.T) {
				expected := testCase.expected.outboundListenAddr
				if actual := resourceConfig.proxyOutboundListenAddr(); expected != actual {
					t.Errorf("Expected: %v Actual: %v", expected, actual)
				}
			})

			t.Run("proxyUID", func(t *testing.T) {
				expected := testCase.expected.proxyUID
				if actual := resourceConfig.proxyUID(); expected != actual {
					t.Errorf("Expected: %v Actual: %v", expected, actual)
				}
			})

			t.Run("proxyLivenessProbe", func(t *testing.T) {
				expected := testCase.expected.livenessProbe
				if actual := resourceConfig.proxyLivenessProbe(); !reflect.DeepEqual(expected, actual) {
					t.Errorf("Expected: %v Actual: %v", expected, actual)
				}
			})

			t.Run("proxyReadinessProbe", func(t *testing.T) {
				expected := testCase.expected.readinessProbe
				if actual := resourceConfig.proxyReadinessProbe(); !reflect.DeepEqual(expected, actual) {
					t.Errorf("Expected: %v Actual: %v", expected, actual)
				}
			})

			t.Run("proxyDestinationProfileSuffixes", func(t *testing.T) {
				expected := testCase.expected.destinationProfileSuffixes
				if actual := resourceConfig.proxyDestinationProfileSuffixes(); expected != actual {
					t.Errorf("Expected: %v Actual: %v", expected, actual)
				}
			})

			t.Run("proxyInitImage", func(t *testing.T) {
				expected := testCase.expected.initImage
				if actual := resourceConfig.proxyInitImage(); expected != actual {
					t.Errorf("Expected: %v Actual: %v", expected, actual)
				}
			})

			t.Run("proxyInitImagePullPolicy", func(t *testing.T) {
				expected := testCase.expected.initImagePullPolicy
				if actual := resourceConfig.proxyInitImagePullPolicy(); expected != actual {
					t.Errorf("Expected: %v Actual: %v", expected, actual)
				}
			})

			t.Run("proxyInitArgs", func(t *testing.T) {
				expected := testCase.expected.initArgs
				if actual := resourceConfig.proxyInitArgs(); !reflect.DeepEqual(expected, actual) {
					t.Errorf("Expected: %v Actual: %v", expected, actual)
				}
			})

			t.Run("proxyInboundSkipPorts", func(t *testing.T) {
				expected := testCase.expected.inboundSkipPorts
				if actual := resourceConfig.proxyInboundSkipPorts(); expected != actual {
					t.Errorf("Expected: %v Actual: %v", expected, actual)
				}
			})

			t.Run("proxyOutboundSkipPorts", func(t *testing.T) {
				expected := testCase.expected.outboundSkipPorts
				if actual := resourceConfig.proxyOutboundSkipPorts(); expected != actual {
					t.Errorf("Expected: %v Actual: %v", expected, actual)
				}
			})
		})
	}
}

func TestProxyInitResourceRequirments(t *testing.T) {
	var (
		resourceConfig = NewResourceConfig(nil, OriginCLI)
		actual         = resourceConfig.proxyInitResourceRequirements()
	)

	expectedLimits := map[corev1.ResourceName]string{
		corev1.ResourceCPU:    proxyInitResourceLimitCPU,
		corev1.ResourceMemory: proxyInitResourceLimitMemory,
	}

	for kind, value := range expectedLimits {
		expected := k8sResource.MustParse(value)
		if v := actual.Limits[kind]; !reflect.DeepEqual(expected, v) {
			t.Errorf("Resource mismatch. Expected %+v. Actual %+v", expected, v)
		}
	}

	expectedRequests := map[corev1.ResourceName]string{
		corev1.ResourceCPU:    proxyInitResourceRequestCPU,
		corev1.ResourceMemory: proxyInitResourceRequestMemory,
	}

	for kind, value := range expectedRequests {
		expected := k8sResource.MustParse(value)
		if v := actual.Requests[kind]; !reflect.DeepEqual(expected, v) {
			t.Errorf("Resource mismatch. Expected %+v. Actual %+v", expected, v)
		}
	}
}

func TestInjectPodSpec(t *testing.T) {
	var (
		configs = &config.All{}
		conf    = NewResourceConfig(configs, OriginUnknown)
	)
	conf.pod.meta = &metav1.ObjectMeta{}
	conf.pod.meta.Annotations = map[string]string{}
	conf.pod.spec = &corev1.PodSpec{}

	t.Run("debug container", func(t *testing.T) {
		patch := NewPatch("Deployment")
		conf.AppendPodAnnotation(k8s.ProxyEnableDebugAnnotation, "true")
		conf.injectPodAnnotations(patch)
		conf.injectPodSpec(patch)

		passed := false
		for _, actual := range patch.patchOps {
			if actual.Op == "add" && actual.Path == "/spec/template/spec/containers/-" {
				container, ok := actual.Value.(*corev1.Container)
				if !ok {
					t.Fatal("Unexpected type assertion error")
				}

				if container.Name == k8s.DebugSidecarName {
					passed = true
					break
				}
			}
		}

		if !passed {
			t.Errorf("Expected debug container to be added to patch. Actual patch: %v", patch.patchOps)
		}
	})

	t.Run("proxy and proxy-init security context", func(t *testing.T) {
		// expect the proxy and proxy-init containers to share the same 'Add' and
		// 'Drop' rules
		testContainer := corev1.Container{
			Name: "test-svc",
			SecurityContext: &corev1.SecurityContext{
				Capabilities: &corev1.Capabilities{
					Add:  []corev1.Capability{"SYS_TIME"},
					Drop: []corev1.Capability{"SYS_ADMIN"},
				},
			},
		}
		conf.pod.spec = &corev1.PodSpec{
			Containers: []corev1.Container{testContainer},
		}
		patch := NewPatch("Deployment")
		conf.injectPodSpec(patch)

		for _, actual := range patch.patchOps {
			if actual.Op == "add" && (actual.Path == "/spec/template/spec/containers/-" ||
				actual.Path == "/spec/template/spec/initContainers/-") {
				container, ok := actual.Value.(*corev1.Container)
				if !ok {
					t.Fatal("Unexpected type assertion error")
				}

				for _, sidecar := range []string{k8s.ProxyContainerName, k8s.InitContainerName} {
					if container.Name == sidecar {
						t.Run(fmt.Sprintf(container.Name), func(t *testing.T) {
							if sc := container.SecurityContext; sc != nil {
								if *sc.AllowPrivilegeEscalation {
									t.Errorf("Expected %s's 'allowPrivilegeEscalation' to be false", container.Name)
								}

								if !*sc.ReadOnlyRootFilesystem {
									t.Errorf("Expected %s's 'readOnlyRootFilesystem' to be true", container.Name)
								}

								if *sc.RunAsUser != conf.proxyUID() {
									t.Errorf("Expected %s's 'RunAsUser' to be %d", container.Name, conf.proxyUID())
								}

								expectedCapabilities := testContainer.SecurityContext.Capabilities
								if container.Name == k8s.InitContainerName {
									expectedCapabilities.Add = append(expectedCapabilities.Add, proxyInitDefaultCapabilities...)
								}

								if !reflect.DeepEqual(sc.Capabilities, expectedCapabilities) {
									t.Errorf("Mismatch 'Add Capabilities' rules. Expected: %v, Actual: %v",
										expectedCapabilities,
										sc.Capabilities.Add)
								}
							} else {
								t.Errorf("Expected %s security context to be non-empty", container.Name)
							}
						})
					}
				}
			}
		}
	})
}
