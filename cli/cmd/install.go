package cmd

import (
	"bytes"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"strings"

	"github.com/ghodss/yaml"
	"github.com/linkerd/linkerd2/pkg/k8s"
	uuid "github.com/satori/go.uuid"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"

	"k8s.io/helm/pkg/chartutil"
	"k8s.io/helm/pkg/engine"
	"k8s.io/helm/pkg/proto/hapi/chart"
	"k8s.io/helm/pkg/renderutil"
	"k8s.io/helm/pkg/timeconv"
	tversion "k8s.io/helm/pkg/version"
)

type installConfig struct {
	Namespace                        string
	ControllerImage                  string
	WebImage                         string
	PrometheusImage                  string
	PrometheusVolumeName             string
	GrafanaImage                     string
	GrafanaVolumeName                string
	ControllerReplicas               uint
	ImagePullPolicy                  string
	UUID                             string
	CliVersion                       string
	ControllerLogLevel               string
	ControllerComponentLabel         string
	CreatedByAnnotation              string
	ProxyAPIPort                     uint
	EnableTLS                        bool
	TLSTrustAnchorVolumeName         string
	TLSSecretsVolumeName             string
	TLSTrustAnchorConfigMapName      string
	ProxyContainerName               string
	TLSTrustAnchorFileName           string
	TLSCertFileName                  string
	TLSPrivateKeyFileName            string
	TLSTrustAnchorVolumeSpecFileName string
	TLSIdentityVolumeSpecFileName    string
	InboundPort                      uint
	OutboundPort                     uint
	IgnoreInboundPorts               string
	IgnoreOutboundPorts              string
	ProxyAutoInjectEnabled           bool
	ProxyAutoInjectLabel             string
	ProxyUID                         int64
	ProxyMetricsPort                 uint
	ProxyControlPort                 uint
	ProxyInjectorTLSSecret           string
	ProxySpecFileName                string
	ProxyInitSpecFileName            string
	ProxyInitImage                   string
	ProxyImage                       string
	ProxyResourceRequestCPU          string
	ProxyResourceRequestMemory       string
	SingleNamespace                  bool
	EnableHA                         bool
	ControllerUID                    int64
	ProfileSuffixes                  string
	EnableH2Upgrade                  bool
}

type installOptions struct {
	controllerReplicas uint
	controllerLogLevel string
	proxyAutoInject    bool
	singleNamespace    bool
	highAvailability   bool
	controllerUID      int64
	disableH2Upgrade   bool
	*proxyConfigOptions
}

const (
	defaultControllerReplicas       = 1
	defaultHAControllerReplicas     = 3
	prometheusProxyOutboundCapacity = 10000

	baseTemplateName  = "base"
	tlsTemplateName   = "tls"
	proxyInjectorName = "proxy_injector"

	baseTemplatePath          = "../install/base_template.yaml"
	tlsTemplatePath           = "../install/tls_template.yaml"
	proxyInjectorTemplatePath = "../install/proxy_injector_template.yaml"
)

func newInstallOptions() *installOptions {
	return &installOptions{
		controllerReplicas: defaultControllerReplicas,
		controllerLogLevel: "info",
		proxyAutoInject:    false,
		singleNamespace:    false,
		highAvailability:   false,
		controllerUID:      2103,
		disableH2Upgrade:   false,
		proxyConfigOptions: newProxyConfigOptions(),
	}
}

func newCmdInstall() *cobra.Command {
	options := newInstallOptions()

	cmd := &cobra.Command{
		Use:   "install [flags]",
		Short: "Output Kubernetes configs to install Linkerd",
		Long:  "Output Kubernetes configs to install Linkerd.",
		RunE: func(cmd *cobra.Command, args []string) error {
			config, err := validateAndBuildConfig(options)
			if err != nil {
				return err
			}

			return render(*config, os.Stdout, options)
		},
	}

	addProxyConfigFlags(cmd, options.proxyConfigOptions)
	cmd.PersistentFlags().UintVar(&options.controllerReplicas, "controller-replicas", options.controllerReplicas, "Replicas of the controller to deploy")
	cmd.PersistentFlags().StringVar(&options.controllerLogLevel, "controller-log-level", options.controllerLogLevel, "Log level for the controller and web components")
	cmd.PersistentFlags().BoolVar(&options.proxyAutoInject, "proxy-auto-inject", options.proxyAutoInject, "Experimental: Enable proxy sidecar auto-injection webhook (default false)")
	cmd.PersistentFlags().BoolVar(&options.singleNamespace, "single-namespace", options.singleNamespace, "Experimental: Configure the control plane to only operate in the installed namespace (default false)")
	cmd.PersistentFlags().BoolVar(&options.highAvailability, "ha", options.highAvailability, "Experimental: Enable HA deployment config for the control plane")
	cmd.PersistentFlags().Int64Var(&options.controllerUID, "controller-uid", options.controllerUID, "Run the control plane components under this user ID")
	cmd.PersistentFlags().BoolVar(&options.disableH2Upgrade, "disable-h2-upgrade", options.disableH2Upgrade, "Prevents the controller from instructing proxies to perform transparent HTTP/2 ugprading")
	return cmd
}

func validateAndBuildConfig(options *installOptions) (*installConfig, error) {
	if err := options.validate(); err != nil {
		return nil, err
	}

	ignoreInboundPorts := []string{
		fmt.Sprintf("%d", options.proxyControlPort),
		fmt.Sprintf("%d", options.proxyMetricsPort),
	}
	for _, p := range options.ignoreInboundPorts {
		ignoreInboundPorts = append(ignoreInboundPorts, fmt.Sprintf("%d", p))
	}
	ignoreOutboundPorts := []string{}
	for _, p := range options.ignoreOutboundPorts {
		ignoreOutboundPorts = append(ignoreOutboundPorts, fmt.Sprintf("%d", p))
	}

	if options.highAvailability && options.controllerReplicas == defaultControllerReplicas {
		options.controllerReplicas = defaultHAControllerReplicas
	}

	if options.highAvailability && options.proxyCPURequest == "" {
		options.proxyCPURequest = "10m"
	}

	if options.highAvailability && options.proxyMemoryRequest == "" {
		options.proxyMemoryRequest = "20Mi"
	}

	profileSuffixes := "."
	if options.proxyConfigOptions.disableExternalProfiles {
		profileSuffixes = "svc.cluster.local."
	}

	return &installConfig{
		Namespace:                        controlPlaneNamespace,
		ControllerImage:                  fmt.Sprintf("%s/controller:%s", options.dockerRegistry, options.linkerdVersion),
		WebImage:                         fmt.Sprintf("%s/web:%s", options.dockerRegistry, options.linkerdVersion),
		PrometheusImage:                  "prom/prometheus:v2.4.0",
		PrometheusVolumeName:             "data",
		GrafanaImage:                     fmt.Sprintf("%s/grafana:%s", options.dockerRegistry, options.linkerdVersion),
		GrafanaVolumeName:                "data",
		ControllerReplicas:               options.controllerReplicas,
		ImagePullPolicy:                  options.imagePullPolicy,
		UUID:                             uuid.NewV4().String(),
		CliVersion:                       k8s.CreatedByAnnotationValue(),
		ControllerLogLevel:               options.controllerLogLevel,
		ControllerComponentLabel:         k8s.ControllerComponentLabel,
		ControllerUID:                    options.controllerUID,
		CreatedByAnnotation:              k8s.CreatedByAnnotation,
		ProxyAPIPort:                     options.proxyAPIPort,
		EnableTLS:                        options.enableTLS(),
		TLSTrustAnchorVolumeName:         k8s.TLSTrustAnchorVolumeName,
		TLSSecretsVolumeName:             k8s.TLSSecretsVolumeName,
		TLSTrustAnchorConfigMapName:      k8s.TLSTrustAnchorConfigMapName,
		ProxyContainerName:               k8s.ProxyContainerName,
		TLSTrustAnchorFileName:           k8s.TLSTrustAnchorFileName,
		TLSCertFileName:                  k8s.TLSCertFileName,
		TLSPrivateKeyFileName:            k8s.TLSPrivateKeyFileName,
		TLSTrustAnchorVolumeSpecFileName: k8s.TLSTrustAnchorVolumeSpecFileName,
		TLSIdentityVolumeSpecFileName:    k8s.TLSIdentityVolumeSpecFileName,
		InboundPort:                      options.inboundPort,
		OutboundPort:                     options.outboundPort,
		IgnoreInboundPorts:               strings.Join(ignoreInboundPorts, ","),
		IgnoreOutboundPorts:              strings.Join(ignoreOutboundPorts, ","),
		ProxyAutoInjectEnabled:           options.proxyAutoInject,
		ProxyAutoInjectLabel:             k8s.ProxyAutoInjectLabel,
		ProxyUID:                         options.proxyUID,
		ProxyMetricsPort:                 options.proxyMetricsPort,
		ProxyControlPort:                 options.proxyControlPort,
		ProxyInjectorTLSSecret:           k8s.ProxyInjectorTLSSecret,
		ProxySpecFileName:                k8s.ProxySpecFileName,
		ProxyInitSpecFileName:            k8s.ProxyInitSpecFileName,
		ProxyInitImage:                   options.taggedProxyInitImage(),
		ProxyImage:                       options.taggedProxyImage(),
		ProxyResourceRequestCPU:          options.proxyCPURequest,
		ProxyResourceRequestMemory:       options.proxyMemoryRequest,
		SingleNamespace:                  options.singleNamespace,
		EnableHA:                         options.highAvailability,
		ProfileSuffixes:                  profileSuffixes,
		EnableH2Upgrade:                  !options.disableH2Upgrade,
	}, nil
}

func (options *installOptions) validate() error {
	if _, err := log.ParseLevel(options.controllerLogLevel); err != nil {
		return fmt.Errorf("--controller-log-level must be one of: panic, fatal, error, warn, info, debug")
	}

	if options.proxyAutoInject && options.singleNamespace {
		return fmt.Errorf("The --proxy-auto-inject and --single-namespace flags cannot both be specified together")
	}

	return options.proxyConfigOptions.validate()
}

func render(config installConfig, w io.Writer, options *installOptions) error {
	// Load chart for rendering
	chrt, err := loadChart(&config)
	if err != nil {
		return err
	}

	// Render raw values
	values, err := renderRawValues(chrt, &config)
	if err != nil {
		return err
	}

	// Render templates with raw values
	renderer := engine.New()
	rendered, err := renderer.Render(chrt, values)
	if err != nil {
		return err
	}

	injectTmpl, err := renderInjectTemplate(rendered, &config)
	if err != nil {
		return err
	}

	injectOptions := newInjectOptions()
	injectOptions.proxyConfigOptions = options.proxyConfigOptions

	// Special case for linkerd-proxy running in the Prometheus pod.
	injectOptions.proxyOutboundCapacity[config.PrometheusImage] = prometheusProxyOutboundCapacity

	return InjectYAML(&injectTmpl, w, ioutil.Discard, injectOptions)
}

func loadChart(config *installConfig) (*chart.Chart, error) {
	chrt := &chart.Chart{}

	// Load chart metadata
	if err := loadMetaData(chrt); err != nil {
		return &chart.Chart{}, err
	}

	// Always load the base template
	if err := loadTemplate(chrt, baseTemplateName, baseTemplatePath); err != nil {
		return &chart.Chart{}, err
	}

	if config.EnableTLS {
		if err := loadTemplate(chrt, tlsTemplateName, tlsTemplatePath); err != nil {
			return &chart.Chart{}, err
		}

		if config.ProxyAutoInjectEnabled {
			if err := loadTemplate(chrt, proxyInjectorName, proxyInjectorTemplatePath); err != nil {
				return &chart.Chart{}, err
			}
		}
	}

	return chrt, nil
}

func loadMetaData(chrt *chart.Chart) error {
	metadataBytes, err := ioutil.ReadFile("../install/Chart.yaml")
	if err != nil {
		return err
	}
	metadata, err := chartutil.UnmarshalChartfile(metadataBytes)
	if err != nil {
		return err
	}
	chrt.Metadata = metadata

	return nil
}

func loadTemplate(chrt *chart.Chart, tmplName string, tmplPath string) error {
	tmpl, err := ioutil.ReadFile(tmplPath)
	if err != nil {
		return err
	}
	chrt.Templates = append(chrt.Templates, &chart.Template{Name: tmplName, Data: tmpl})

	return nil
}

func renderRawValues(chrt *chart.Chart, config *installConfig) (chartutil.Values, error) {
	// Initialize helper structs for use in rendering values
	caps := &chartutil.Capabilities{
		APIVersions:   chartutil.DefaultVersionSet,
		KubeVersion:   chartutil.DefaultKubeVersion,
		TillerVersion: tversion.GetVersionProto(),
	}

	renderOpts := renderutil.Options{
		ReleaseOptions: chartutil.ReleaseOptions{
			Name:      "release-name",
			IsInstall: true,
			IsUpgrade: false,
			Time:      timeconv.Now(),
			Namespace: "default",
		},
		KubeVersion: "",
	}

	rawValues, err := yaml.Marshal(config)
	if err != nil {
		return chartutil.Values{}, err
	}

	chrtConfig := &chart.Config{Raw: string(rawValues), Values: map[string]*chart.Value{}}

	return chartutil.ToRenderValuesCaps(chrt, chrtConfig, renderOpts.ReleaseOptions, caps)
}

func renderInjectTemplate(rendered map[string]string, config *installConfig) (bytes.Buffer, error) {
	var injectTmpl bytes.Buffer

	// Order matters in template; the base template must be first
	if _, err := injectTmpl.WriteString(rendered["linkerd/base"]); err != nil {
		return bytes.Buffer{}, err
	}

	// If TLS should be enabled, then the TLS template must be second
	if config.EnableTLS {
		if _, err := injectTmpl.WriteString(rendered["linkerd/tls"]); err != nil {
			return bytes.Buffer{}, err
		}

		// If TLS is enabled and the proxy should be injected, then the proxy
		// injector template should be third
		if config.ProxyAutoInjectEnabled {
			if _, err := injectTmpl.WriteString(rendered["linkerd/proxy_injector"]); err != nil {
				return bytes.Buffer{}, err
			}
		}
	}

	return injectTmpl, nil
}
