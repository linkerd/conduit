package cmd

import (
	"fmt"
	"regexp"
	"time"

	"github.com/runconduit/conduit/controller/api/public"
	pb "github.com/runconduit/conduit/controller/gen/public"
	"github.com/runconduit/conduit/pkg/k8s"
	"github.com/runconduit/conduit/pkg/version"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

var controlPlaneNamespace string
var apiAddr string // An empty value means "use the Kubernetes configuration"
var kubeconfigPath string
var verbose bool

var RootCmd = &cobra.Command{
	Use:   "conduit",
	Short: "conduit manages the Conduit service mesh",
	Long:  `conduit manages the Conduit service mesh.`,
	PersistentPreRun: func(cmd *cobra.Command, args []string) {
		// enable / disable logging
		if verbose {
			log.SetLevel(log.DebugLevel)
		} else {
			log.SetLevel(log.PanicLevel)
		}
	},
}

func init() {
	RootCmd.PersistentFlags().StringVarP(&controlPlaneNamespace, "conduit-namespace", "c", "conduit", "Namespace in which Conduit is installed")
	RootCmd.PersistentFlags().StringVar(&kubeconfigPath, "kubeconfig", "", "Path to the kubeconfig file to use for CLI requests")
	RootCmd.PersistentFlags().StringVar(&apiAddr, "api-addr", "", "Override kubeconfig and communicate directly with the control plane at host:port (mostly for testing)")
	RootCmd.PersistentFlags().BoolVar(&verbose, "verbose", false, "Turn on debug logging")

	RootCmd.AddCommand(newCmdCheck())
	RootCmd.AddCommand(newCmdCompletion())
	RootCmd.AddCommand(newCmdDashboard())
	RootCmd.AddCommand(newCmdGet())
	RootCmd.AddCommand(newCmdInject())
	RootCmd.AddCommand(newCmdInstall())
	RootCmd.AddCommand(newCmdStat())
	RootCmd.AddCommand(newCmdTap())
	RootCmd.AddCommand(newCmdVersion())
}

func newPublicAPIClient() (pb.ApiClient, error) {
	if apiAddr != "" {
		return public.NewInternalClient(apiAddr)
	}
	kubeAPI, err := k8s.NewAPI(kubeconfigPath)
	if err != nil {
		return nil, err
	}
	return public.NewExternalClient(controlPlaneNamespace, kubeAPI)
}

type proxyConfigOptions struct {
	conduitVersion   string
	proxyImage       string
	initImage        string
	imagePullPolicy  string
	proxyUID         int64
	proxyLogLevel    string
	proxyBindTimeout string
	proxyAPIPort     uint
	proxyControlPort uint
	proxyMetricsPort uint
	tls              string
}

func newProxyConfigOptions() *proxyConfigOptions {
	return &proxyConfigOptions{
		conduitVersion:   version.Version,
		proxyImage:       "gcr.io/runconduit/proxy",
		initImage:        "gcr.io/runconduit/proxy-init",
		imagePullPolicy:  "IfNotPresent",
		proxyUID:         2102,
		proxyLogLevel:    "warn,conduit_proxy=info",
		proxyBindTimeout: "10s",
		proxyAPIPort:     8086,
		proxyControlPort: 4190,
		proxyMetricsPort: 4191,
		tls:              "",
	}
}

const optionalTLS = "optional"

var alphaNumDashDot = regexp.MustCompile("^[\\.a-zA-Z0-9-]+$")

func (options *proxyConfigOptions) validate() error {
	if !alphaNumDashDot.MatchString(options.conduitVersion) {
		return fmt.Errorf("%s is not a valid version", options.conduitVersion)
	}
	if options.imagePullPolicy != "Always" && options.imagePullPolicy != "IfNotPresent" && options.imagePullPolicy != "Never" {
		return fmt.Errorf("--image-pull-policy must be one of: Always, IfNotPresent, Never")
	}
	if _, err := time.ParseDuration(options.proxyBindTimeout); err != nil {
		return fmt.Errorf("Invalid duration '%s' for --proxy-bind-timeout flag", options.proxyBindTimeout)
	}
	if options.tls != "" && options.tls != optionalTLS {
		return fmt.Errorf("--tls must be blank or set to \"%s\"", optionalTLS)
	}
	return nil
}

func (options *proxyConfigOptions) enableTLS() bool {
	return options.tls == optionalTLS
}

func addProxyConfigFlags(cmd *cobra.Command, options *proxyConfigOptions) {
	cmd.PersistentFlags().StringVarP(&options.conduitVersion, "conduit-version", "v", options.conduitVersion, "Tag to be used for Conduit images")
	cmd.PersistentFlags().StringVar(&options.initImage, "init-image", options.initImage, "Conduit init container image name")
	cmd.PersistentFlags().StringVar(&options.proxyImage, "proxy-image", options.proxyImage, "Conduit proxy container image name")
	cmd.PersistentFlags().StringVar(&options.imagePullPolicy, "image-pull-policy", options.imagePullPolicy, "Docker image pull policy")
	cmd.PersistentFlags().Int64Var(&options.proxyUID, "proxy-uid", options.proxyUID, "Run the proxy under this user ID")
	cmd.PersistentFlags().StringVar(&options.proxyLogLevel, "proxy-log-level", options.proxyLogLevel, "Log level for the proxy")
	cmd.PersistentFlags().StringVar(&options.proxyBindTimeout, "proxy-bind-timeout", options.proxyBindTimeout, "Timeout the proxy will use")
	cmd.PersistentFlags().UintVar(&options.proxyAPIPort, "api-port", options.proxyAPIPort, "Port where the Conduit controller is running")
	cmd.PersistentFlags().UintVar(&options.proxyControlPort, "control-port", options.proxyControlPort, "Proxy port to use for control")
	cmd.PersistentFlags().UintVar(&options.proxyMetricsPort, "metrics-port", options.proxyMetricsPort, "Proxy port to serve metrics on")
	cmd.PersistentFlags().StringVar(&options.tls, "tls", options.tls, "Enable TLS; valid settings: \"optional\"")
	cmd.PersistentFlags().MarkHidden("tls")
}
