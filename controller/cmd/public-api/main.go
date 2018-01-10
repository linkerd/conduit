package main

import (
	"context"
	"flag"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	log "github.com/sirupsen/logrus"

	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/runconduit/conduit/controller/api/public"
	"github.com/runconduit/conduit/controller/tap"
	"github.com/runconduit/conduit/controller/telemetry"
)

func main() {
	addr := flag.String("addr", ":8085", "address to serve on")
	metricsAddr := flag.String("metrics-addr", ":9995", "address to serve scrapable metrics on")
	telemetryAddr := flag.String("telemetry-addr", ":8087", "address of telemetry service")
	tapAddr := flag.String("tap-addr", ":8088", "address of tap service")
	flag.Parse()

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt, syscall.SIGTERM)

	telemetryClient, telemetryConn, err := telemetry.NewClient(*telemetryAddr)
	if err != nil {
		log.Fatal(err.Error())
	}
	defer telemetryConn.Close()

	tapClient, tapConn, err := tap.NewClient(*tapAddr)
	if err != nil {
		log.Fatal(err.Error())
	}
	defer tapConn.Close()

	server := public.NewServer(*addr, telemetryClient, tapClient)

	go func() {
		log.Infof("starting HTTP server on %+v", *addr)
		server.ListenAndServe()
	}()

	go func() {
		log.Infof("serving scrapable metrics on %+v", *metricsAddr)
		http.Handle("/metrics", promhttp.Handler())
		http.ListenAndServe(*metricsAddr, nil)
	}()

	<-stop

	log.Infof("shutting down HTTP server on %+v", *addr)
	server.Shutdown(context.Background())
}
