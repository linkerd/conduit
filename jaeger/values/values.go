package values

import (
	"fmt"

	"github.com/linkerd/linkerd2/jaeger/static"
	"github.com/linkerd/linkerd2/pkg/charts"
	l5dcharts "github.com/linkerd/linkerd2/pkg/charts/linkerd2"
	"helm.sh/helm/v3/pkg/chart/loader"
	"helm.sh/helm/v3/pkg/chartutil"
	"sigs.k8s.io/yaml"
)

// Values represents the values of jaeger template
type Values struct {
	Namespace string    `json:"namespace"`
	Collector collector `json:"collector"`
	Jaeger    jaeger    `json:"jaeger"`
}

type collector struct {
	Resources l5dcharts.Resources `json:"resources"`
	Image     l5dcharts.Image     `json:"image"`
}

type jaeger struct {
	Resources l5dcharts.Resources `json:"resources"`
	Image     l5dcharts.Image     `json:"image"`
}

// NewValues returns a new instance of the Values type.
// TODO: Add HA logic
func NewValues() (*Values, error) {
	chartDir := fmt.Sprintf("%s/", "jaeger")
	v, err := readDefaults(chartDir)
	if err != nil {
		return nil, err
	}

	return v, nil
}

// readDefaults read all the default variables from the values.yaml file.
// chartDir is the root directory of the Helm chart where values.yaml is.
func readDefaults(chartDir string) (*Values, error) {
	valuesFile := &loader.BufferedFile{
		Name: chartutil.ValuesfileName,
	}

	if err := charts.ReadFile(static.Templates, chartDir, valuesFile); err != nil {
		return nil, err
	}

	var values Values
	if err := yaml.Unmarshal(valuesFile.Data, &values); err != nil {
		return nil, err
	}

	return &values, nil
}
