package test

import (
	"fmt"
	"os"
	"regexp"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/linkerd/linkerd2/testutil"
)

type deploySpec struct {
	replicas   int
	containers []string
}

const (
	proxyContainer = "linkerd-proxy"
)

//////////////////////
///   TEST SETUP   ///
//////////////////////

var TestHelper *testutil.TestHelper

func TestMain(m *testing.M) {
	TestHelper = testutil.NewTestHelper()
	os.Exit(m.Run())
}

var (
	linkerdSvcs = []string{
		"linkerd-controller-api",
		"linkerd-grafana",
		"linkerd-prometheus",
		"linkerd-destination",
		"linkerd-web",
	}

	linkerdDeployReplicas = map[string]deploySpec{
		"linkerd-controller": {1, []string{"destination", "public-api", "tap"}},
		"linkerd-grafana":    {1, []string{}},
		"linkerd-prometheus": {1, []string{}},
		"linkerd-web":        {1, []string{"web"}},
	}

	// Linkerd commonly logs these errors during testing, remove these once
	// they're addressed.
	// TODO: eliminate these errors: https://github.com/linkerd/linkerd2/issues/2453
	knownErrorsRegex = regexp.MustCompile(strings.Join([]string{
		// TLS not ready at startup
		`.*-tls linkerd-(ca|controller|grafana|prometheus|web)-.*-.* linkerd-proxy ERR! admin={bg=tls-config} linkerd2_proxy::transport::tls::config error loading /var/linkerd-io/identity/certificate\.crt: No such file or directory \(os error 2\)`,
		`.*-tls linkerd-(ca|controller|grafana|prometheus|web)-.*-.* linkerd-proxy ERR! admin={bg=tls-config} linkerd2_proxy::transport::tls::config error loading /var/linkerd-io/trust-anchors/trust-anchors\.pem: No such file or directory \(os error 2\)`,
		`.*-tls linkerd-(ca|controller|grafana|prometheus|web)-.*-.* linkerd-proxy WARN admin={bg=tls-config} linkerd2_proxy::transport::tls::config error reloading TLS config: Io\("/var/linkerd-io/identity/certificate\.crt", Some\(2\)\), falling back`,
		`.*-tls linkerd-(ca|controller|grafana|prometheus|web)-.*-.* linkerd-proxy WARN admin={bg=tls-config} linkerd2_proxy::transport::tls::config error reloading TLS config: Io\("/var/linkerd-io/trust-anchors/trust-anchors\.pem", Some\(2\)\), falling back`,

		`.*-tls linkerd-(ca|controller|grafana|prometheus|web)-.*-.* linkerd-proxy WARN proxy={server=in listen=0.0.0.0:4143} rustls::session Sending fatal alert AccessDenied`,

		// k8s hitting readiness endpoints before components are ready
		`.* linkerd-(ca|controller|grafana|prometheus|web)-.*-.* linkerd-proxy ERR! proxy={server=in listen=0\.0\.0\.0:4143 remote=.*} linkerd2_proxy::proxy::http::router service error: an error occurred trying to connect: Connection refused \(os error 111\) \(address: 127\.0\.0\.1:.*\)`,
		`.* linkerd-(ca|controller|grafana|prometheus|web)-.*-.* linkerd-proxy ERR! proxy={server=out listen=127\.0\.0\.1:4140 remote=.*} linkerd2_proxy::proxy::http::router service error: an error occurred trying to connect: Connection refused \(os error 111\) \(address: .*:4191\)`,

		`.* linkerd-(ca|controller|grafana|prometheus|web)-.*-.* linkerd-proxy ERR! admin={server=metrics listen=0.0.0.0:4191 remote=.*} linkerd2_proxy::control::serve_http error serving metrics: Error { kind: Shutdown, cause: Os { code: 107, kind: NotConnected, message: "Transport endpoint is not connected" } }`,

		`.* linkerd-controller-.*-.* tap time=".*" level=error msg="\[.*\] encountered an error: rpc error: code = Canceled desc = context canceled"`,
		`.* linkerd-web-.*-.* linkerd-proxy WARN trust_dns_proto::xfer::dns_exchange failed to associate send_message response to the sender`,

		// prometheus scrape failures of control-plane
		`.* linkerd-prometheus-.*-.* linkerd-proxy ERR! proxy={server=out listen=127\.0\.0\.1:4140 remote=.*} linkerd2_proxy::proxy::http::router service error: an error occurred trying to connect: Connection refused \(os error 111\) \(address: .*:(3000|999(4|5|6|7|8))\)`,
		`.* linkerd-prometheus-.*-.* linkerd-proxy ERR! proxy={server=out listen=127\.0\.0\.1:4140 remote=.*} linkerd2_proxy::proxy::http::router service error: an error occurred trying to connect: operation timed out after 300ms`,

		// single namespace warnings
		`.*-single-namespace linkerd-controller-.*-.* (destination|public-api|tap) time=".*" level=warning msg="Not authorized for cluster-wide access, limiting access to \\".*-single-namespace\\" namespace"`,
		`.*-single-namespace linkerd-controller-.*-.* (destination|public-api|tap) time=".*" level=warning msg="ServiceProfiles not available"`,

		`.* linkerd-web-.*-.* web time=".*" level=error msg="Post http://linkerd-controller-api\..*\.svc\.cluster\.local:8085/api/v1/Version: context canceled"`,
		`.*-tls linkerd-web-.*-.* linkerd-proxy ERR! linkerd-destination\..*-tls\.svc\.cluster\.local:8086 rustls::session TLS alert received: Message {`,
		`.*-tls linkerd-controller-.*-.* linkerd-proxy ERR! .*:9090 rustls::session TLS alert received: Message {`,
		`.*-tls linkerd-web-.*-.* linkerd-proxy WARN linkerd-destination\..*-tls\.svc\.cluster\.local:8086 linkerd2_proxy::proxy::reconnect connect error to Config { addr: Name\(NameAddr { name: "linkerd-destination\..*-tls\.svc\.cluster\.local", port: 8086 }\), tls_server_identity: Some\("linkerd-controller\.deployment\..*-tls\.linkerd-managed\..*-tls.svc.cluster.local"\), tls_config: Some\(ClientConfig\) }: received fatal alert: AccessDenied`,
		`.*-tls linkerd-controller-.*-.* linkerd-proxy WARN .*:9090 linkerd2_proxy::proxy::reconnect connect error to Config { target: Target { addr: V4\(.*:9090\), tls: Some\(ConnectionConfig { server_identity: "linkerd-prometheus\.deployment\..*-tls.linkerd-managed\..*-tls\.svc\.cluster\.local", config: ClientConfig }\) }, settings: Http2, _p: \(\) }: received fatal alert: AccessDenied`,
	}, "|"))
)

//////////////////////
/// TEST EXECUTION ///
//////////////////////

// Tests are executed in serial in the order defined
// Later tests depend on the success of earlier tests

func TestVersionPreInstall(t *testing.T) {
	err := TestHelper.CheckVersion("unavailable")
	if err != nil {
		t.Fatalf("Version command failed\n%s", err.Error())
	}
}

func TestCheckPreInstall(t *testing.T) {
	cmd := []string{"check", "--pre", "--expected-version", TestHelper.GetVersion()}
	golden := "check.pre.golden"
	if TestHelper.SingleNamespace() {
		cmd = append(cmd, "--single-namespace")
		golden = "check.pre.single_namespace.golden"
		err := TestHelper.CreateNamespaceIfNotExists(TestHelper.GetLinkerdNamespace())
		if err != nil {
			t.Fatalf("Namespace creation failed\n%s", err.Error())
		}
	}
	out, _, err := TestHelper.LinkerdRun(cmd...)
	if err != nil {
		t.Fatalf("Check command failed\n%s", out)
	}

	err = TestHelper.ValidateOutput(out, golden)
	if err != nil {
		t.Fatalf("Received unexpected output\n%s", err.Error())
	}
}

func TestInstall(t *testing.T) {
	cmd := []string{"install",
		"--controller-log-level", "debug",
		"--proxy-log-level", "warn,linkerd2_proxy=debug",
		"--linkerd-version", TestHelper.GetVersion(),
	}
	if TestHelper.TLS() {
		cmd = append(cmd, []string{"--tls", "optional"}...)
		linkerdDeployReplicas["linkerd-ca"] = deploySpec{1, []string{"ca"}}
	}
	if TestHelper.SingleNamespace() {
		cmd = append(cmd, "--single-namespace")
	}

	out, _, err := TestHelper.LinkerdRun(cmd...)
	if err != nil {
		t.Fatalf("linkerd install command failed\n%s", out)
	}

	out, err = TestHelper.KubectlApply(out, TestHelper.GetLinkerdNamespace())
	if err != nil {
		t.Fatalf("kubectl apply command failed\n%s", out)
	}

	// Tests Namespace
	err = TestHelper.CheckIfNamespaceExists(TestHelper.GetLinkerdNamespace())
	if err != nil {
		t.Fatalf("Received unexpected output\n%s", err.Error())
	}

	// Tests Services
	for _, svc := range linkerdSvcs {
		if err := TestHelper.CheckService(TestHelper.GetLinkerdNamespace(), svc); err != nil {
			t.Error(fmt.Errorf("Error validating service [%s]:\n%s", svc, err))
		}
	}

	// Tests Pods and Deployments
	for deploy, spec := range linkerdDeployReplicas {
		if err := TestHelper.CheckPods(TestHelper.GetLinkerdNamespace(), deploy, spec.replicas); err != nil {
			t.Fatal(fmt.Errorf("Error validating pods for deploy [%s]:\n%s", deploy, err))
		}
		if err := TestHelper.CheckDeployment(TestHelper.GetLinkerdNamespace(), deploy, spec.replicas); err != nil {
			t.Fatal(fmt.Errorf("Error validating deploy [%s]:\n%s", deploy, err))
		}
	}
}

func TestVersionPostInstall(t *testing.T) {
	err := TestHelper.CheckVersion(TestHelper.GetVersion())
	if err != nil {
		t.Fatalf("Version command failed\n%s", err.Error())
	}
}

func TestCheckPostInstall(t *testing.T) {
	cmd := []string{"check", "--expected-version", TestHelper.GetVersion(), "--wait=0"}
	golden := "check.golden"
	if TestHelper.SingleNamespace() {
		cmd = append(cmd, "--single-namespace")
		golden = "check.single_namespace.golden"
	}

	err := TestHelper.RetryFor(time.Minute, func() error {
		out, _, err := TestHelper.LinkerdRun(cmd...)

		if err != nil {
			return fmt.Errorf("Check command failed\n%s", out)
		}

		err = TestHelper.ValidateOutput(out, golden)
		if err != nil {
			return fmt.Errorf("Received unexpected output\n%s", err.Error())
		}

		return nil
	})
	if err != nil {
		t.Fatal(err.Error())
	}
}

func TestDashboard(t *testing.T) {
	dashboardPort := 52237
	dashboardURL := fmt.Sprintf("http://127.0.0.1:%d", dashboardPort)

	outputStream, err := TestHelper.LinkerdRunStream("dashboard", "-p",
		strconv.Itoa(dashboardPort), "--show", "url")
	if err != nil {
		t.Fatalf("Error running command:\n%s", err)
	}
	defer outputStream.Stop()

	outputLines, err := outputStream.ReadUntil(4, 1*time.Minute)
	if err != nil {
		t.Fatalf("Error running command:\n%s", err)
	}

	output := strings.Join(outputLines, "")
	if !strings.Contains(output, dashboardURL) {
		t.Fatalf("Dashboard command failed. Expected url [%s] not present", dashboardURL)
	}

	resp, err := TestHelper.HTTPGetURL(dashboardURL + "/api/version")
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	if !strings.Contains(resp, TestHelper.GetVersion()) {
		t.Fatalf("Dashboard command failed. Expected response [%s] to contain version [%s]",
			resp, TestHelper.GetVersion())
	}
}

func TestInject(t *testing.T) {
	cmd := []string{"inject", "testdata/smoke_test.yaml"}
	if TestHelper.TLS() {
		cmd = append(cmd, []string{"--tls", "optional"}...)
	}

	out, injectReport, err := TestHelper.LinkerdRun(cmd...)
	if err != nil {
		t.Fatalf("linkerd inject command failed\n%s", out)
	}

	err = TestHelper.ValidateOutput(injectReport, "inject.report.golden")
	if err != nil {
		t.Fatalf("Received unexpected output\n%s", err.Error())
	}

	prefixedNs := TestHelper.GetTestNamespace("smoke-test")
	out, err = TestHelper.KubectlApply(out, prefixedNs)
	if err != nil {
		t.Fatalf("kubectl apply command failed\n%s", out)
	}

	for _, deploy := range []string{"smoke-test-terminus", "smoke-test-gateway"} {
		err = TestHelper.CheckPods(prefixedNs, deploy, 1)
		if err != nil {
			t.Fatalf("Unexpected error: %v", err)
		}
	}

	url, err := TestHelper.URLFor(prefixedNs, "smoke-test-gateway", 8080)
	if err != nil {
		t.Fatalf("Failed to get URL: %s", err)
	}

	output, err := TestHelper.HTTPGetURL(url)
	if err != nil {
		t.Fatalf("Unexpected error: %v %s", err, output)
	}

	expectedStringInPayload := "\"payload\":\"BANANA\""
	if !strings.Contains(output, expectedStringInPayload) {
		t.Fatalf("Expected application response to contain string [%s], but it was [%s]",
			expectedStringInPayload, output)
	}
}

func TestCheckProxy(t *testing.T) {
	prefixedNs := TestHelper.GetTestNamespace("smoke-test")
	cmd := []string{"check", "--proxy", "--expected-version", TestHelper.GetVersion(), "--namespace", prefixedNs, "--wait=0"}
	golden := "check.proxy.golden"
	if TestHelper.SingleNamespace() {
		cmd = append(cmd, "--single-namespace")
		golden = "check.proxy.single_namespace.golden"
	}

	err := TestHelper.RetryFor(time.Minute, func() error {
		out, _, err := TestHelper.LinkerdRun(cmd...)
		if err != nil {
			return fmt.Errorf("Check command failed\n%s", out)
		}

		err = TestHelper.ValidateOutput(out, golden)
		if err != nil {
			return fmt.Errorf("Received unexpected output\n%s", err.Error())
		}

		return nil
	})
	if err != nil {
		t.Fatal(err.Error())
	}
}

func TestLogs(t *testing.T) {
	controllerRegex := regexp.MustCompile("level=(panic|fatal|error|warn)")
	proxyRegex := regexp.MustCompile(fmt.Sprintf("%s (ERR|WARN)", proxyContainer))

	for deploy, spec := range linkerdDeployReplicas {
		deploy := strings.TrimPrefix(deploy, "linkerd-")
		containers := append(spec.containers, proxyContainer)

		for _, container := range containers {
			errRegex := controllerRegex
			if container == proxyContainer {
				errRegex = proxyRegex
			}

			outputStream, err := TestHelper.LinkerdRunStream(
				"logs", "--no-color",
				"--control-plane-component", deploy,
				"--container", container,
			)
			if err != nil {
				t.Errorf("Error running command:\n%s", err)
			}
			defer outputStream.Stop()
			// Ignore the error returned, since ReadUntil will return an error if it
			// does not return 10,000 after 1 second. We don't need 10,000 log lines.
			outputLines, _ := outputStream.ReadUntil(10000, 2*time.Second)
			if len(outputLines) == 0 {
				t.Errorf("No logs found for %s/%s", deploy, container)
			}

			for _, line := range outputLines {
				if errRegex.MatchString(line) && !knownErrorsRegex.MatchString(line) {
					t.Errorf("Found error in %s/%s log: %s", deploy, container, line)
				}
			}
		}
	}
}

func TestRestarts(t *testing.T) {
	for deploy, spec := range linkerdDeployReplicas {
		if err := TestHelper.CheckPods(TestHelper.GetLinkerdNamespace(), deploy, spec.replicas); err != nil {
			t.Fatal(fmt.Errorf("Error validating pods [%s]:\n%s", deploy, err))
		}
	}
}
