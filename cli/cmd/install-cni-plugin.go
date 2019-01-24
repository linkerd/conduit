package cmd

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"strings"
	"text/template"

	"github.com/linkerd/linkerd2/cli/install"
	"github.com/linkerd/linkerd2/pkg/version"
	"github.com/spf13/cobra"
)

type installCNIPluginConfig struct {
	Namespace           string
	CNIPluginImage      string
	LogLevel            string
	InboundPort         uint
	OutboundPort        uint
	IgnoreInboundPorts  string
	IgnoreOutboundPorts string
	ProxyUID            int64
}

type cniPluginOptions struct {
	linkerdVersion      string
	dockerRegistry      string
	proxyControlPort    uint
	proxyMetricsPort    uint
	inboundPort         uint
	outboundPort        uint
	ignoreInboundPorts  []uint
	ignoreOutboundPorts []uint
	proxyUID            int64
	cniPluginImage      string
	logLevel            string
}

func newCNIPluginOptions() *cniPluginOptions {
	return &cniPluginOptions{
		linkerdVersion:      version.Version,
		dockerRegistry:      defaultDockerRegistry,
		proxyControlPort:    4190,
		proxyMetricsPort:    4191,
		inboundPort:         4143,
		outboundPort:        4140,
		ignoreInboundPorts:  nil,
		ignoreOutboundPorts: nil,
		proxyUID:            2102,
		cniPluginImage:      defaultDockerRegistry + "/cni-plugin",
		logLevel:            "info",
	}
}

func (options *cniPluginOptions) validate() error {
	if !alphaNumDashDot.MatchString(options.linkerdVersion) {
		return fmt.Errorf("%s is not a valid version", options.linkerdVersion)
	}

	if !alphaNumDashDotSlashColon.MatchString(options.dockerRegistry) {
		return fmt.Errorf("%s is not a valid Docker registry. The url can contain only letters, numbers, dash, dot, slash and colon", options.dockerRegistry)
	}

	found := false
	allowedLogLevels := [...]string{"trace", "debug", "info", "warn", "error", "fatal", "panic"}
	for _, level := range allowedLogLevels {
		if level == options.logLevel {
			found = true
		}
	}
	if !found {
		fmt.Printf("%s is not a valid log level for the linkerd-cni plugin so defaulting to warn. Must be one of [trace, debug, info, warn, error, fatal, panic]", options.dockerRegistry)
	}

	return nil
}

func (options *cniPluginOptions) taggedCNIPluginImage() string {
	image := strings.Replace(options.cniPluginImage, defaultDockerRegistry, options.dockerRegistry, 1)
	return fmt.Sprintf("%s:%s", image, options.linkerdVersion)
}

func newCmdInstallCNIPlugin() *cobra.Command {
	options := newCNIPluginOptions()

	cmd := &cobra.Command{
		Use:   "install-cni [flags]",
		Short: "Output Kubernetes configs to install the Linkerd CNI Plugin",
		Long: `Output Kubernetes configs to install the Linkerd CNI Plugin."

This command installs a DaemonSet into the Linkerd control plane. The DaemonSet
copies the necessary linkerd-cni plugin binaries and configs onto the host. It
assumes that the 'linkerd install' command will be executed with the '--no-init-container'
flag. This command needs to be executed before the 'linkerd install --no-init-container'
command.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			config, err := validateAndBuildCNIConfig(options)
			if err != nil {
				return err
			}
			return renderCNIPlugin(os.Stdout, config)
		},
	}

	cmd.PersistentFlags().StringVarP(&options.linkerdVersion, "linkerd-version", "v", options.linkerdVersion, "Tag to be used for Linkerd images")
	cmd.PersistentFlags().StringVar(&options.dockerRegistry, "registry", options.dockerRegistry, "Docker registry to pull images from")
	cmd.PersistentFlags().Int64Var(&options.proxyUID, "proxy-uid", options.proxyUID, "Run the proxy under this user ID")
	cmd.PersistentFlags().UintVar(&options.inboundPort, "inbound-port", options.inboundPort, "Proxy port to use for inbound traffic")
	cmd.PersistentFlags().UintVar(&options.outboundPort, "outbound-port", options.outboundPort, "Proxy port to use for outbound traffic")
	cmd.PersistentFlags().UintVar(&options.proxyControlPort, "control-port", options.proxyControlPort, "Proxy port to use for control")
	cmd.PersistentFlags().UintVar(&options.proxyMetricsPort, "metrics-port", options.proxyMetricsPort, "Proxy port to serve metrics on")
	cmd.PersistentFlags().UintSliceVar(&options.ignoreInboundPorts, "skip-inbound-ports", options.ignoreInboundPorts, "Ports that should skip the proxy and send directly to the application")
	cmd.PersistentFlags().UintSliceVar(&options.ignoreOutboundPorts, "skip-outbound-ports", options.ignoreOutboundPorts, "Outbound ports that should skip the proxy")
	cmd.PersistentFlags().StringVar(&options.cniPluginImage, "cni-image", options.cniPluginImage, "Image for the cni-plugin.")
	cmd.PersistentFlags().StringVar(&options.logLevel, "cni-log-level", options.logLevel, "Log level for the cni-plugin.")

	return cmd
}

func validateAndBuildCNIConfig(options *cniPluginOptions) (*installCNIPluginConfig, error) {
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

	return &installCNIPluginConfig{
		Namespace:           controlPlaneNamespace,
		CNIPluginImage:      options.taggedCNIPluginImage(),
		LogLevel:            options.logLevel,
		InboundPort:         options.inboundPort,
		OutboundPort:        options.outboundPort,
		IgnoreInboundPorts:  strings.Join(ignoreInboundPorts, ","),
		IgnoreOutboundPorts: strings.Join(ignoreOutboundPorts, ","),
		ProxyUID:            options.proxyUID,
	}, nil
}

func renderCNIPlugin(w io.Writer, config *installCNIPluginConfig) error {
	template, err := template.New("linkerd-cni").Parse(install.CNITemplate)
	if err != nil {
		return err
	}
	buf := &bytes.Buffer{}
	err = template.Execute(buf, config)
	if err != nil {
		return err
	}

	w.Write(buf.Bytes())
	w.Write([]byte("---\n"))

	return nil
}
