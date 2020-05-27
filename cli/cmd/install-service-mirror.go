package cmd

import (
	"errors"
	"fmt"
	"io"
	"os"

	"github.com/linkerd/linkerd2/pkg/charts"
	"github.com/linkerd/linkerd2/pkg/charts/servicemirror"
	"github.com/linkerd/linkerd2/pkg/version"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	kerrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/helm/pkg/chartutil"
	"sigs.k8s.io/yaml"
)

type installServiceMirrorOptions struct {
	namespace           string
	controlPlaneVersion string
	dockerRegistry      string
	logLevel            string
	uid                 int64
	requeueLimit        int32
}

const helmServiceMirrorDefaultChartName = "linkerd2-service-mirror"

func newCmdInstallServiceMirror() *cobra.Command {
	options, err := newInstallServiceMirrorOptionsWithDefaults()
	if err != nil {
		fmt.Fprint(os.Stderr, err)
		os.Exit(1)
	}

	cmd := &cobra.Command{
		Use:   "install-service-mirror [flags]",
		Short: "Output Kubernetes configs to install Linkerd Service Mirror",
		Long:  "Output Kubernetes configs to install Linkerd Service Mirror",
		RunE: func(cmd *cobra.Command, args []string) error {
			return renderServiceMirror(os.Stdout, options)
		},
		Hidden: true,
	}

	cmd.PersistentFlags().StringVarP(&options.controlPlaneVersion, "control-plane-version", "", options.controlPlaneVersion, "(Development) Tag to be used for the control plane component images")
	cmd.PersistentFlags().StringVar(&options.dockerRegistry, "registry", options.dockerRegistry, "Docker registry to pull images from")
	cmd.PersistentFlags().StringVarP(&options.logLevel, "log-level", "", options.logLevel, "Log level for the Service Mirror Component")
	cmd.PersistentFlags().Int64Var(&options.uid, "uid", options.uid, "Run the Service Mirror component under this user ID")
	cmd.PersistentFlags().Int32Var(&options.requeueLimit, "event-requeue-limit", options.requeueLimit, "The number of times a failed update from the remote cluster is allowed to be requeued (retried)")
	cmd.PersistentFlags().StringVarP(&options.namespace, "namespace", "", options.namespace, "The namespace in which the Service Mirror Component is to be installed")

	return cmd
}

func newInstallServiceMirrorOptionsWithDefaults() (*installServiceMirrorOptions, error) {
	defaults, err := servicemirror.NewValues()
	if err != nil {
		return nil, err
	}
	return &installServiceMirrorOptions{
		namespace:           defaults.Namespace,
		controlPlaneVersion: version.Version,
		dockerRegistry:      defaultDockerRegistry,
		logLevel:            defaults.LogLevel,
		uid:                 defaults.ServiceMirrorUID,
		requeueLimit:        defaults.EventRequeueLimit,
	}, nil
}

func (options *installServiceMirrorOptions) buildValues() (*servicemirror.Values, error) {
	installValues, err := servicemirror.NewValues()
	if err != nil {
		return nil, err
	}
	installValues.Namespace = options.namespace
	installValues.LogLevel = options.logLevel
	installValues.ControllerImageVersion = options.controlPlaneVersion
	installValues.ControllerImage = fmt.Sprintf("%s/controller", options.dockerRegistry)
	installValues.ServiceMirrorUID = options.uid
	installValues.EventRequeueLimit = options.requeueLimit

	return installValues, nil
}

func (options *installServiceMirrorOptions) validate() error {

	_, err := getLinkerdConfigMap()
	if err != nil {
		if kerrors.IsNotFound(err) {
			return errors.New("you need Linkerd to be installed in order to install the service mirroring component")
		}
		return err
	}

	if !alphaNumDashDot.MatchString(options.controlPlaneVersion) {
		return fmt.Errorf("%s is not a valid version", options.controlPlaneVersion)
	}

	if options.namespace == "" {
		return errors.New("you need to specify a namespace")
	}

	if options.namespace == controlPlaneNamespace {
		return errors.New("you need to install the service mirror component in a namespace different than the Linkerd one")
	}

	if _, err := log.ParseLevel(options.logLevel); err != nil {
		return fmt.Errorf("--log-level must be one of: panic, fatal, error, warn, info, debug")
	}

	return nil
}

func renderServiceMirror(w io.Writer, config *installServiceMirrorOptions) error {
	if err := config.validate(); err != nil {
		return err
	}

	values, err := config.buildValues()
	if err != nil {
		return err
	}

	// Render raw values and create chart config
	rawValues, err := yaml.Marshal(values)
	if err != nil {
		return err
	}

	files := []*chartutil.BufferedFile{
		{Name: chartutil.ChartfileName},
		{Name: "templates/service-mirror.yaml"},
	}

	chart := &charts.Chart{
		Name:      helmServiceMirrorDefaultChartName,
		Dir:       helmServiceMirrorDefaultChartName,
		Namespace: controlPlaneNamespace,
		RawValues: rawValues,
		Files:     files,
	}
	buf, err := chart.RenderNoPartials()
	if err != nil {
		return err
	}
	w.Write(buf.Bytes())
	w.Write([]byte("---\n"))

	return nil

}
