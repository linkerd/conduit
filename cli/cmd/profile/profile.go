package profile

import (
	"errors"
	"fmt"
	"os"

	"github.com/linkerd/linkerd2/pkg/profiles"
	"github.com/spf13/cobra"
	"k8s.io/apimachinery/pkg/util/validation"
)

type templateConfig struct {
	ControlPlaneNamespace string
	ServiceNamespace      string
	ServiceName           string
	ClusterZone           string
}

type profileOptions struct {
	name      string
	namespace string
	template  bool
	openAPI   string
	proto     string
}

func newProfileOptions() *profileOptions {
	return &profileOptions{
		name:      "",
		namespace: "default",
		template:  false,
		openAPI:   "",
		proto:     "",
	}
}

func (options *profileOptions) validate() error {
	outputs := 0
	if options.template {
		outputs++
	}
	if options.openAPI != "" {
		outputs++
	}
	if outputs != 1 {
		return errors.New("You must specify exactly one of --template or --open-api")
	}

	// a DNS-1035 label must consist of lower case alphanumeric characters or '-',
	// start with an alphabetic character, and end with an alphanumeric character
	if errs := validation.IsDNS1035Label(options.name); len(errs) != 0 {
		return fmt.Errorf("invalid service %q: %v", options.name, errs)
	}

	// a DNS-1123 label must consist of lower case alphanumeric characters or '-',
	// and must start and end with an alphanumeric character
	if errs := validation.IsDNS1123Label(options.namespace); len(errs) != 0 {
		return fmt.Errorf("invalid namespace %q: %v", options.namespace, errs)
	}

	return nil
}

// NewCmdProfile creates a new cobra command for the Profile subcommand which
// generates Linkerd service profiles.
func NewCmdProfile(controlPlaneNamespace string) *cobra.Command {

	options := newProfileOptions()

	cmd := &cobra.Command{
		Use:   "profile [flags] (--template | --open-api file) (SERVICE)",
		Short: "Output service profile config for Kubernetes",
		Long: `Output service profile config for Kubernetes.

This outputs a service profile for the given service.

If the --template flag is specified, it outputs a service profile template.
Edit the template and then apply it with kubectl to add a service profile to
a service.

Example:
  linkerd profile -n emojivoto --template web-svc > web-svc-profile.yaml
  # (edit web-svc-profile.yaml manually)
  kubectl apply -f web-svc-profile.yaml

If the --open-api flag is specified, it reads the given OpenAPI
specification file and outputs a corresponding service profile.

Example:
  linkerd profile -n emojivoto --open-api web-svc.swagger web-svc | kubectl apply -f -`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			options.name = args[0]

			err := options.validate()
			if err != nil {
				return err
			}

			if options.template {
				return profiles.RenderProfileTemplate(options.namespace, options.name, controlPlaneNamespace, os.Stdout)
			} else if options.openAPI != "" {
				return renderOpenAPI(options, controlPlaneNamespace, os.Stdout)
			}

			// we should never get here
			return errors.New("Unexpected error")
		},
	}

	cmd.PersistentFlags().BoolVar(&options.template, "template", options.template, "Output a service profile template")
	cmd.PersistentFlags().StringVar(&options.openAPI, "open-api", options.openAPI, "Output a service profile based on the given OpenAPI spec file")
	cmd.PersistentFlags().StringVarP(&options.namespace, "namespace", "n", options.namespace, "Namespace of the service")

	return cmd
}
