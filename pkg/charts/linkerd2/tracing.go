package linkerd2

import (
	"k8s.io/helm/pkg/chartutil"
	"sigs.k8s.io/yaml"
)

var (
	tracingAddOn = "tracing"
)

// Tracing is an add-on that installs the distributed tracing
// related components like OpenCensus Collector and Jaeger
type Tracing map[string]interface{}

// Name returns the name of the Tracing add-on
func (t Tracing) Name() string {
	return tracingAddOn
}

// Values returns the configuration values that were assigned for this add-on
func (t Tracing) Values() []byte {
	values, err := yaml.Marshal(t)
	if err != nil {
		return nil
	}
	return values
}

// Templates returns the template files specific to this add-on
func (t Tracing) Templates() []*chartutil.BufferedFile {
	return []*chartutil.BufferedFile{
		{Name: chartutil.ChartfileName},
		{Name: "templates/tracing-rbac.yaml"},
		{Name: "templates/tracing.yaml"},
	}
}
