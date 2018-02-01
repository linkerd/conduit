package cmd

import (
	"fmt"
	"os"

	"github.com/pkg/browser"
	"github.com/runconduit/conduit/pkg/k8s"
	"github.com/runconduit/conduit/pkg/shell"
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
			return fmt.Errorf("port must be positive, was %d", proxyPort)
		}

		kubectl, err := k8s.NewKubectl(shell.NewUnixShell())
		if err != nil {
			fmt.Fprintf(os.Stderr, "Failed to start kubectl: %s", err)
			os.Exit(1)
		}

		asyncProcessErr := make(chan error, 1)

		err = kubectl.StartProxy(asyncProcessErr, proxyPort)

		if err != nil {
			fmt.Fprintf(os.Stderr, "Failed to start kubectl proxy: %s", err)
			os.Exit(1)
		}

		url, err := kubectl.UrlFor(controlPlaneNamespace, "/services/web:http/proxy/")

		if err != nil {
			fmt.Fprintf(os.Stderr, "Failed to generate URL for dashboard: %s", err)
			os.Exit(1)
		}

		fmt.Printf("Opening [%s] in the default browser\n", url)
		err = browser.OpenURL(url.String())

		if err != nil {
			fmt.Fprintf(os.Stderr, "Failed to open URL %s in the default browser: %s", url, err)
			os.Exit(1)
		}

		select {
		case err = <-asyncProcessErr:
			if err != nil {
				fmt.Fprintf(os.Stderr, "Error starting proxy via kubectl: %s", err)
				os.Exit(1)
			}
		}
		close(asyncProcessErr)
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
