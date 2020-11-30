package tracing

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/linkerd/linkerd2/testutil"
)

type (
	traces struct {
		Data []trace `json:"data"`
	}

	trace struct {
		Processes map[string]process `json:"processes"`
	}

	process struct {
		ServiceName string `json:"serviceName"`
	}
)

//////////////////////
///   TEST SETUP   ///
//////////////////////

var TestHelper *testutil.TestHelper

func TestMain(m *testing.M) {
	TestHelper = testutil.NewTestHelper()
	os.Exit(m.Run())
}

//////////////////////
/// TEST EXECUTION ///
//////////////////////

func TestTracing(t *testing.T) {

	ctx := context.Background()
	if os.Getenv("RUN_ARM_TEST") != "" {
		t.Skip("Skipped. Jaeger & Open Census images does not support ARM yet")
	}

	// linkerd-jaeger extension
	tracingNs := "linkerd-jaeger"
	out, err := TestHelper.LinkerdRun("jaeger", "install")
	if err != nil {
		testutil.AnnotatedFatal(t, "'linkerd jaeger install' command failed", err)
	}

	out, err = TestHelper.KubectlApply(out, "")
	if err != nil {
		testutil.AnnotatedFatalf(t, "'kubectl apply' command failed",
			"'kubectl apply' command failed\n%s", out)
	}

	// Emojivoto components
	emojivotoNs := TestHelper.GetTestNamespace("emojivoto")
	err = TestHelper.CreateDataPlaneNamespaceIfNotExists(ctx, emojivotoNs, nil)
	if err != nil {
		testutil.AnnotatedFatalf(t, fmt.Sprintf("failed to create %s namespace", emojivotoNs),
			"failed to create %s namespace: %s", emojivotoNs, err)
	}

	emojivotoYaml, err := testutil.ReadFile("testdata/emojivoto.yaml")
	if err != nil {
		testutil.AnnotatedFatalf(t, "failed to read emojivoto yaml",
			"failed to read emojivoto yaml\n%s\n", err)
	}
	emojivotoYaml = strings.ReplaceAll(emojivotoYaml, "___TRACING_NS___", tracingNs)
	out, stderr, err := TestHelper.PipeToLinkerdRun(emojivotoYaml, "inject", "-")
	if err != nil {
		testutil.AnnotatedFatalf(t, "'linkerd inject' command failed",
			"'linkerd inject' command failed\n%s\n%s", out, stderr)
	}

	out, err = TestHelper.KubectlApply(out, emojivotoNs)
	if err != nil {
		testutil.AnnotatedFatalf(t, "'kubectl apply' command failed",
			"'kubectl apply' command failed\n%s", out)
	}

	// Ingress components
	// Ingress must run in the same namespace as the service it routes to (web)
	ingressNs := emojivotoNs
	err = TestHelper.CreateDataPlaneNamespaceIfNotExists(ctx, ingressNs, nil)
	if err != nil {
		testutil.AnnotatedFatalf(t, fmt.Sprintf("failed to create %s namespace", ingressNs),
			"failed to create %s namespace: %s", ingressNs, err)
	}

	ingressYaml, err := testutil.ReadFile("testdata/ingress.yaml")
	if err != nil {
		testutil.AnnotatedFatalf(t, "failed to read ingress yaml",
			"failed to read ingress yaml\n%s\n", err)
	}
	ingressYaml = strings.ReplaceAll(ingressYaml, "___INGRESS_NAMESPACE___", ingressNs)
	ingressYaml = strings.ReplaceAll(ingressYaml, "___TRACING_NS___", tracingNs)
	out, stderr, err = TestHelper.PipeToLinkerdRun(ingressYaml, "inject", "-")
	if err != nil {
		testutil.AnnotatedFatalf(t, "'linkerd inject' command failed",
			"'linkerd inject' command failed\n%s\n%s", out, stderr)
	}

	out, err = TestHelper.KubectlApply(out, ingressNs)
	if err != nil {
		testutil.AnnotatedFatalf(t, "'kubectl apply' command failed",
			"'kubectl apply' command failed\n%s", out)
	}

	// wait for deployments to start
	for _, deploy := range []struct {
		ns   string
		name string
	}{
		{ns: emojivotoNs, name: "vote-bot"},
		{ns: emojivotoNs, name: "web"},
		{ns: emojivotoNs, name: "emoji"},
		{ns: emojivotoNs, name: "voting"},
		{ns: ingressNs, name: "nginx-ingress"},
		{ns: tracingNs, name: "collector"},
		{ns: tracingNs, name: "jaeger"},
	} {
		if err := TestHelper.CheckPods(ctx, deploy.ns, deploy.name, 1); err != nil {
			if rce, ok := err.(*testutil.RestartCountError); ok {
				testutil.AnnotatedWarn(t, "CheckPods timed-out", rce)
			} else {
				testutil.AnnotatedError(t, "CheckPods timed-out", err)
			}
		}

		if err := TestHelper.CheckDeployment(ctx, deploy.ns, deploy.name, 1); err != nil {
			testutil.AnnotatedErrorf(t, "CheckDeployment timed-out", "Error validating deployment [%s]:\n%s", deploy.name, err)
		}
	}

	t.Run("expect full trace", func(t *testing.T) {

		url, err := TestHelper.URLFor(ctx, tracingNs, "jaeger", 16686)
		if err != nil {
			testutil.AnnotatedFatal(t, "error building URL", err)
		}
		timeout := 120 * time.Second
		err = TestHelper.RetryFor(timeout, func() error {
			tracesJSON, err := TestHelper.HTTPGetURL(url + "/api/traces?lookback=1h&service=nginx")
			if err != nil {
				return err
			}
			traces := traces{}

			err = json.Unmarshal([]byte(tracesJSON), &traces)
			if err != nil {
				return err
			}

			processes := []string{"nginx", "web", "emoji", "voting", "linkerd-proxy"}
			if !hasTraceWithProcesses(&traces, processes) {
				return fmt.Errorf("No trace found with processes: %s", processes)
			}
			return nil
		})
		if err != nil {
			testutil.AnnotatedFatal(t, fmt.Sprintf("timed-out checking trace (%s)", timeout), err)
		}
	})
}

func hasTraceWithProcesses(traces *traces, ps []string) bool {
	for _, trace := range traces.Data {
		if containsProcesses(trace, ps) {
			return true
		}
	}
	return false
}

func containsProcesses(trace trace, ps []string) bool {
	toFind := make(map[string]struct{})
	for _, p := range ps {
		toFind[p] = struct{}{}
	}
	for _, p := range trace.Processes {
		delete(toFind, p.ServiceName)
	}
	return len(toFind) == 0
}
