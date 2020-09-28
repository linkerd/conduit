package webhook

import (
	"context"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/linkerd/linkerd2/controller/k8s"
	"github.com/linkerd/linkerd2/pkg/admin"
	"github.com/linkerd/linkerd2/pkg/flags"
	pkgk8s "github.com/linkerd/linkerd2/pkg/k8s"
	"github.com/linkerd/linkerd2/pkg/tls"
	log "github.com/sirupsen/logrus"
)

// Launch sets up and starts the webhook and metrics servers
func Launch(ctx context.Context, APIResources []k8s.APIResource, metricsPort uint32, handler handlerFunc, component, subcommand string, args []string) {
	cmd := flag.NewFlagSet(subcommand, flag.ExitOnError)

	metricsAddr := cmd.String("metrics-addr", fmt.Sprintf(":%d", metricsPort), "address to serve scrapable metrics on")
	addr := cmd.String("addr", ":8443", "address to serve on")
	kubeconfig := cmd.String("kubeconfig", "", "path to kubeconfig")

	flags.ConfigureAndParse(cmd, args)

	stop := make(chan os.Signal, 1)
	defer close(stop)
	signal.Notify(stop, os.Interrupt, syscall.SIGTERM)

	k8sAPI, err := k8s.InitializeAPI(ctx, *kubeconfig, true, APIResources...)
	if err != nil {
		log.Fatalf("failed to initialize Kubernetes API: %s", err)
	}

	cred, err := tls.ReadPEMCreds(pkgk8s.MountPathTLSKeyPEM, pkgk8s.MountPathTLSCrtPEM)
	if err != nil {
		log.Fatalf("failed to read TLS secrets: %s", err)
	}

	s, err := NewServer(k8sAPI, *addr, cred, handler, component)
	if err != nil {
		log.Fatalf("failed to initialize the webhook server: %s", err)
	}

	k8sAPI.Sync(nil)

	go s.Start()
	go admin.StartServer(*metricsAddr)

	<-stop
	log.Info("shutting down webhook server")
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	if err := s.Shutdown(ctx); err != nil {
		log.Error(err)
	}
}
