package egress

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"testing"

	"github.com/runconduit/conduit/testutil"
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

func TestEgressHttp(t *testing.T) {
	out, err := TestHelper.ConduitRun("inject", "testdata/proxy.yaml")
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	prefixedNs := TestHelper.GetTestNamespace("egress-test")
	out, err = TestHelper.KubectlApply(out, prefixedNs)
	if err != nil {
		t.Fatalf("Unexpected error: %v output:\n%s", err, out)
	}

	test_case := func(serviceName, dnsName, protocolToUse, methodToUse string) {
		testName := fmt.Sprintf("Can use egress to send %s request to %s (%s)", methodToUse, protocolToUse, serviceName)
		t.Run(testName, func(t *testing.T) {
			expectedURL := fmt.Sprintf("%s://%s/%s", protocolToUse, dnsName, strings.ToLower(methodToUse))

			svcURL, err := TestHelper.GetURLForService(prefixedNs, serviceName)
			if err != nil {
				t.Fatalf("Failed to get service URL: %v", err)
			}

			output, err := TestHelper.HTTPGetURL(svcURL)
			if err != nil {
				t.Fatalf("Unexpected error: %v", err)
			}
			var jsonResponse map[string]interface{}
			json.Unmarshal([]byte(output), &jsonResponse)

			payloadText := jsonResponse["payload"]
			if payloadText == nil {
				t.Fatalf("Expected [%s] request to [%s] to return a payload, got nil. Response:\n%s\n", methodToUse, expectedURL, output)
			}

			var messagePayload map[string]interface{}
			json.Unmarshal([]byte(payloadText.(string)), &messagePayload)

			actualURL := messagePayload["url"]
			if actualURL != expectedURL {
				t.Fatalf("Expecting response to say egress sent [%s] request to URL [%s] but got [%s]. Response:\n%s\n", methodToUse, expectedURL, actualURL, output)
			}
		})
	}

	supportedProtocols := []string{"http", "https"}
	methods := []string{"GET", "POST"}
	for _, protocolToUse := range supportedProtocols {
		for _, methodToUse := range methods {
			serviceName := fmt.Sprintf("egress-test-%s-%s-svc", protocolToUse, strings.ToLower(methodToUse))
			test_case(serviceName, "www.httpbin.org", protocolToUse, methodToUse)
		}
	}

	// Test egress for a domain with fewer than 3 segments.
	test_case("egress-test-not-www-get-svc", "httpbin.org", "https", "GET")
}
