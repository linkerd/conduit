package version

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"time"

	healthcheckPb "github.com/runconduit/conduit/controller/gen/common/healthcheck"
	pb "github.com/runconduit/conduit/controller/gen/public"
	"github.com/runconduit/conduit/pkg/healthcheck"
	log "github.com/sirupsen/logrus"
)

// DO NOT EDIT
// This var is updated automatically as part of the build process
var Version = "undefined"

const (
	VersionSubsystemName         = "conduit-version"
	CliCheckDescription          = "cli is up-to-date"
	ControlPlaneCheckDescription = "control plane is up-to-date"
)

func VersionFlag() *bool {
	return flag.Bool("version", false, "print version and exit")
}

func MaybePrintVersionAndExit(printVersion bool) {
	if printVersion {
		fmt.Println(Version)
		os.Exit(0)
	}
	log.Infof("running conduit version %s", Version)
}

var httpClientTimeout = 10 * time.Second

type versionStatusChecker struct {
	version         string
	versionCheckURL string
	versionOverride string
	publicApiClient pb.ApiClient
	httpClient      http.Client
}

func (v versionStatusChecker) SelfCheck() []*healthcheckPb.CheckResult {
	cliVersion := v.version
	cliIsUpToDate := &healthcheckPb.CheckResult{
		Status:           healthcheckPb.CheckStatus_OK,
		SubsystemName:    VersionSubsystemName,
		CheckDescription: CliCheckDescription,
	}

	latestVersion, err := v.getLatestVersion()
	if err != nil {
		cliIsUpToDate.Status = healthcheckPb.CheckStatus_ERROR
		cliIsUpToDate.FriendlyMessageToUser = err.Error()
		return []*healthcheckPb.CheckResult{cliIsUpToDate}
	}
	if cliVersion != latestVersion {
		cliIsUpToDate.Status = healthcheckPb.CheckStatus_FAIL
		cliIsUpToDate.FriendlyMessageToUser = fmt.Sprintf("is running version %s but the latest version is %s", cliVersion, latestVersion)
	}

	controlPlaneIsUpToDate := &healthcheckPb.CheckResult{
		Status:           healthcheckPb.CheckStatus_OK,
		SubsystemName:    VersionSubsystemName,
		CheckDescription: ControlPlaneCheckDescription,
	}

	controlPlaneVersion, err := v.getServerVersion()
	if err != nil {
		controlPlaneIsUpToDate.Status = healthcheckPb.CheckStatus_ERROR
		controlPlaneIsUpToDate.FriendlyMessageToUser = err.Error()
		return []*healthcheckPb.CheckResult{controlPlaneIsUpToDate}
	}
	if controlPlaneVersion != latestVersion {
		controlPlaneIsUpToDate.Status = healthcheckPb.CheckStatus_FAIL
		controlPlaneIsUpToDate.FriendlyMessageToUser = fmt.Sprintf("is running version %s but the latest version is %s", controlPlaneVersion, latestVersion)
	}

	checks := []*healthcheckPb.CheckResult{cliIsUpToDate}
	checks = append(checks, controlPlaneIsUpToDate)
	return checks
}

func (v versionStatusChecker) getServerVersion() (string, error) {
	resp, err := v.publicApiClient.Version(context.Background(), &pb.Empty{})
	if err != nil {
		return "", err
	}

	return resp.GetReleaseVersion(), nil
}

func (v versionStatusChecker) getLatestVersion() (string, error) {
	if v.versionOverride != "" {
		return v.versionOverride, nil
	}

	resp, err := v.httpClient.Get(v.versionCheckURL)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return "", fmt.Errorf("got %d error from %s", resp.StatusCode, v.versionCheckURL)
	}

	bodyBytes, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	var l map[string]string
	err = json.Unmarshal(bodyBytes, &l)
	if err != nil {
		return "", err
	}

	return l["version"], nil
}

func NewVersionStatusChecker(versionCheckURL, versionOverride string, client pb.ApiClient) healthcheck.StatusChecker {
	return versionStatusChecker{
		version:         Version,
		versionCheckURL: versionCheckURL,
		versionOverride: versionOverride,
		publicApiClient: client,
		httpClient:      http.Client{Timeout: httpClientTimeout},
	}
}
