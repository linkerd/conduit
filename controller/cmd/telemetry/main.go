package main

import (
	"flag"
	"os"
	"os/signal"
	"strings"
	"syscall"

	"github.com/runconduit/conduit/controller/telemetry"
	"github.com/runconduit/conduit/controller/util"
	"github.com/runconduit/conduit/pkg/version"
	log "github.com/sirupsen/logrus"
)

func main() {
	addr := flag.String("addr", "127.0.0.1:8087", "address to serve on")
	metricsAddr := flag.String("metrics-addr", ":9997", "address to serve scrapable metrics on")
	prometheusUrl := flag.String("prometheus-url", "http://127.0.0.1:9090", "prometheus url")
	controllerNamespace := flag.String("controller-namespace", "conduit", "namespace in which Conduit is installed")
	ignoredNamespaces := flag.String("ignore-namespaces", "", "comma separated list of namespaces to not list pods from")
	kubeConfigPath := flag.String("kubeconfig", "", "path to kube config")
	logLevel := flag.String("log-level", log.InfoLevel.String(), "log level, must be one of: panic, fatal, error, warn, info, debug")
	printVersion := version.VersionFlag()
	flag.Parse()

	// set global log level
	level, err := log.ParseLevel(*logLevel)
	if err != nil {
		log.Fatalf("invalid log-level: %s", *logLevel)
	}
	log.SetLevel(level)

	version.MaybePrintVersionAndExit(*printVersion)

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt, syscall.SIGTERM)

	server, lis, err := telemetry.NewServer(*addr, *controllerNamespace, *prometheusUrl,
		strings.Split(*ignoredNamespaces, ","), *kubeConfigPath)
	if err != nil {
		log.Fatal(err.Error())
	}

	go func() {
		log.Println("starting gRPC server on", *addr)
		server.Serve(lis)
	}()

	go util.NewMetricsServer(*metricsAddr)

	<-stop

	log.Println("shutting down gRPC server on", *addr)
	server.GracefulStop()
}
