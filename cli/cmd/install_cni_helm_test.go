package cmd

import (
	"bytes"
	"path/filepath"
	"testing"

	cnicharts "github.com/linkerd/linkerd2/pkg/charts/cni"
	"k8s.io/helm/pkg/chartutil"
	pb "k8s.io/helm/pkg/proto/hapi/chart"
	"k8s.io/helm/pkg/renderutil"
	"sigs.k8s.io/yaml"
)

func TestRenderCniHelm(t *testing.T) {
	// read the cni plugin chart and its defaults from the local folder.
	// override most defaults with pinned values.
	// use the Helm lib to render the templates.
	// the golden file is generated using the following `helm template` command:
	// bin/helm template --set namespace="linkerd-test" --set controllerNamespaceLabel="linkerd.io/control-plane-ns-test" --set cniResourceAnnotation="linkerd.io/cni-resource-test" --set inboundProxyPort=1234 --set outboundProxyPort=5678 --set createdByAnnotation="linkerd.io/created-by-test" --set cniPluginImage="gcr.io/linkerd-io/cni-plugin-test" --set cniPluginVersion="test-version" --set logLevel="debug" --set proxyUID=1111 --set destCNINetDir="/etc/cni/net.d-test" --set destCNIBinDir="/opt/cni/bin-test" --set useWaitFlag=true --set cliVersion=test-version charts/linkerd2-cni

	t.Run("Cni Install", func(t *testing.T) {
		chartCni := chartCniPlugin(t)
		testRenderCniHelm(t, chartCni, "install_cni_helm_output.golden")
	})

}

func testRenderCniHelm(t *testing.T, chart *pb.Chart, goldenFileName string) {
	var (
		chartName = "linkerd2-cni"
		namespace = "linkerd-test"
	)
	overrideJSON :=
		`{
			"namespace": "linkerd-test",
  			"cniResourceLabel": "linkerd.io/cni-resource-test",
  			"inboundProxyPort": 1234,
  			"outboundProxyPort": 5678,
			"createdByAnnotation": "linkerd.io/created-by-test",
  			"cniPluginImage": "gcr.io/linkerd-io/cni-plugin-test",
  			"cniPluginVersion": "test-version",
  			"logLevel": "debug",
  			"proxyUID": 1111,
  			"destCNINetDir": "/etc/cni/net.d-test",
  			"destCNIBinDir": "/opt/cni/bin-test",
  			"useWaitFlag": true,
			"cliVersion": "test-version"
		}`

	overrideConfig := &pb.Config{Raw: overrideJSON}

	releaseOptions := renderutil.Options{
		ReleaseOptions: chartutil.ReleaseOptions{
			Name:      chartName,
			Namespace: namespace,
			IsUpgrade: false,
			IsInstall: true,
		},
	}

	rendered, err := renderutil.Render(chart, overrideConfig, releaseOptions)
	if err != nil {
		t.Fatal("Unexpected error", err)
	}

	var buf bytes.Buffer
	for _, template := range chart.Templates {
		source := chartName + "/" + template.Name
		v, exists := rendered[source]
		if !exists {
			// skip partial templates
			continue
		}
		buf.WriteString("---\n# Source: " + source + "\n")
		buf.WriteString(v)
	}

	diffTestdata(t, goldenFileName, buf.String())
}

func chartCniPlugin(t *testing.T) *pb.Chart {
	values, err := readCniTestValues(t)
	if err != nil {
		t.Fatal("Unexpected error", err)
	}

	chartPartials := chartPartials(t, []string{"templates/_helpers.tpl"})

	chart := &pb.Chart{
		Metadata: &pb.Metadata{
			Name: helmCNIDefaultChartName,
			Sources: []string{
				filepath.Join("..", "..", "..", "charts", "linkerd2-cni"),
			},
		},
		Dependencies: []*pb.Chart{
			chartPartials,
		},
		Values: &pb.Config{
			Raw: string(values),
		},
	}

	chart.Templates = append(chart.Templates, &pb.Template{
		Name: "templates/cni-plugin.yaml",
	})

	for _, template := range chart.Templates {
		filepath := filepath.Join(chart.Metadata.Sources[0], template.Name)
		template.Data = []byte(readTestdata(t, filepath))
	}

	return chart
}

func readCniTestValues(t *testing.T) ([]byte, error) {
	values, err := cnicharts.NewValues()
	if err != nil {
		t.Fatal("Unexpected error", err)
	}

	return yaml.Marshal(values)
}
