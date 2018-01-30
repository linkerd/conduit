package cmd

import (
	"fmt"
	"os"

	"github.com/runconduit/conduit/controller/api/public"
	pb "github.com/runconduit/conduit/controller/gen/public"
	"github.com/runconduit/conduit/pkg/k8s"
	"github.com/runconduit/conduit/pkg/shell"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

var cfgFile string
var controlPlaneNamespace string
var apiAddr string // An empty value means "use the Kubernetes configuration"
var kubeconfigPath string
var verbose bool

var RootCmd = &cobra.Command{
	Use:   "conduit",
	Short: "conduit manages the Conduit service mesh",
	Long:  `conduit manages the Conduit service mesh.`,
	PersistentPreRun: func(cmd *cobra.Command, args []string) {
		// turn on debug logging
		if verbose {
			log.SetLevel(log.DebugLevel)
		}
	},
}

func init() {
	RootCmd.PersistentFlags().StringVarP(&controlPlaneNamespace, "conduit-namespace", "n", "conduit", "namespace in which Conduit is installed")
	RootCmd.PersistentFlags().BoolVar(&verbose, "verbose", false, "turn on debug logging")
}

// TODO: decide if we want to use viper

func addControlPlaneNetworkingArgs(cmd *cobra.Command) {
	// Use the same argument name as `kubectl` (see the output of `kubectl options`).
	//TODO: move these to init() as they are globally applicable
	cmd.PersistentFlags().StringVar(&kubeconfigPath, "kubeconfig", "", "Path to the kubeconfig file to use for CLI requests")
	cmd.PersistentFlags().StringVar(&apiAddr, "api-addr", "", "Override kubeconfig and communicate directly with the control plane at host:port (mostly for testing)")
}

func newPublicAPIClient() (pb.ApiClient, error) {
	if apiAddr != "" {
		return public.NewInternalClient(apiAddr)
	}
	kubeApi, err := k8s.NewK8sAPI(shell.NewUnixShell(), kubeconfigPath)
	if err != nil {
		return nil, err
	}
	return public.NewExternalClient(controlPlaneNamespace, kubeApi)
}

// This is equivalent to calling log.Fatalf, but unlike go's logger interface,
// it does not support any formatting of the output prior to printing, which is
// preferable for printing errors from the CLI
func logFatalf(format string, v ...interface{}) {
	fmt.Fprintf(os.Stderr, format+"\n", v...)
	os.Exit(1)
}
