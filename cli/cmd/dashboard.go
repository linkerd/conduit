package cmd

import (
	"fmt"

	"github.com/pkg/browser"
	"github.com/runconduit/conduit/cli/dashboard"
	"github.com/runconduit/conduit/pkg/k8s"
	"github.com/runconduit/conduit/pkg/shell"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

var (
	proxyPort = -1
)

var dashboardCmd = &cobra.Command{
	Use:   "dashboard [flags]",
	Short: "Open the Conduit dashboard in a web browser",
	Long:  "Open the Conduit dashboard in a web browser.",
	RunE: func(cmd *cobra.Command, args []string) error {
		if proxyPort <= 0 {
			log.Fatalf("port must be positive, was %d", proxyPort)
		}

		shell := shell.NewUnixShell()
		kubectl, err := k8s.NewKubectl(shell)

		if err != nil {
			return fmt.Errorf("failed to start kubectl: %v", err)
		}

		kubeApi, err := k8s.NewK8sAPI(shell, kubeconfigPath)
		if err != nil {
			return fmt.Errorf("failed to connect to the Kubernetes API: %v", err)
		}

		dshBoard := dashboard.NewDashboardHandler(kubectl, kubeApi)

		if dshBoard.IsDashboardAvailable() {
			asyncProcessErr := make(chan error, 1)

			err = dshBoard.StartProxy(asyncProcessErr, proxyPort)
			if err != nil {
				log.Fatalf("Failed to start kubectl proxy: %v", err)
			}

			url, err := dshBoard.UrlFor(controlPlaneNamespace, "/services/web:http/proxy/")
			if err != nil {
				log.Fatalf("Failed to generate URL for dashboard: %v", err)
			}

			fmt.Printf("Opening [%s] in the default browser\n", url)

			err = browser.OpenURL(url.String())
			if err != nil {
				log.Fatalf("failed to open URL %s in the default browser: %v", url, err)
			}

			select {
			case err = <-asyncProcessErr:
				if err != nil {
					log.Fatalf("Error starting proxy via kubectl: %v", err)
				}
			}

			close(asyncProcessErr)

			return nil
		}
		log.Fatal("Failed to access conduit dashboard")
		return nil
	},
}

func init() {
	RootCmd.AddCommand(dashboardCmd)
	dashboardCmd.Args = cobra.NoArgs

	// This is identical to what `kubectl proxy --help` reports, except
	// `kubectl proxy` allows `--port=0` to indicate a random port; That's
	// inconvenient to support so it isn't supported.
	dashboardCmd.PersistentFlags().IntVarP(&proxyPort, "port", "p", 8001, "The port on which to run the proxy, which must not be 0.")
}
