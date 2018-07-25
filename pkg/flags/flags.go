package flags

import (
	"flag"
	"fmt"
	"os"

	"github.com/linkerd/linkerd2/pkg/version"
	log "github.com/sirupsen/logrus"
)

// ConfigureAndParse adds flags that are common to all go processes, and
// overrides the default flag for glog logging, which we can't disable. This
// func calls flag.Parse(), so it should be called after all other flags have
// been configured.
func ConfigureAndParse() {
	// override glog's default configuration
	flag.Set("logtostderr", "true")

	logLevel := flag.String("log-level", log.InfoLevel.String(),
		"log level, must be one of: panic, fatal, error, warn, info, debug")
	printVersion := flag.Bool("version", false, "print version and exit")

	flag.Parse()

	setLogLevel(*logLevel)
	maybePrintVersionAndExit(*printVersion)
}

func setLogLevel(logLevel string) {
	level, err := log.ParseLevel(logLevel)
	if err != nil {
		log.Fatalf("invalid log-level: %s", logLevel)
	}
	log.SetLevel(level)
}

func maybePrintVersionAndExit(printVersion bool) {
	if printVersion {
		fmt.Println(version.Version)
		os.Exit(0)
	}
	log.Infof("running version %s", version.Version)
}
