package charts

import (
	"bytes"
	"path"
	"strings"

	"github.com/linkerd/linkerd2/pkg/charts/static"
	"github.com/linkerd/linkerd2/pkg/version"
	"k8s.io/helm/pkg/chartutil"
	helmChart "k8s.io/helm/pkg/proto/hapi/chart"
	"k8s.io/helm/pkg/renderutil"
	"k8s.io/helm/pkg/timeconv"
)

const versionPlaceholder = "linkerdVersionValue"

// Chart holds the necessary info to render a Helm chart
type Chart struct {
	Name      string
	Dir       string
	Namespace string
	RawValues []byte
	Files     []*chartutil.BufferedFile
}

func (c *Chart) render(partialsFiles []*chartutil.BufferedFile) (bytes.Buffer, error) {
	if err := FilesReader(c.Dir+"/", c.Files); err != nil {
		return bytes.Buffer{}, err
	}

	if err := FilesReader("", partialsFiles); err != nil {
		return bytes.Buffer{}, err
	}

	// Create chart and render templates
	chart, err := chartutil.LoadFiles(append(c.Files, partialsFiles...))
	if err != nil {
		return bytes.Buffer{}, err
	}

	renderOpts := renderutil.Options{
		ReleaseOptions: chartutil.ReleaseOptions{
			Name:      c.Name,
			IsInstall: true,
			IsUpgrade: false,
			Time:      timeconv.Now(),
			Namespace: c.Namespace,
		},
		KubeVersion: "",
	}

	chartConfig := &helmChart.Config{Raw: string(c.RawValues), Values: map[string]*helmChart.Value{}}
	renderedTemplates, err := renderutil.Render(chart, chartConfig, renderOpts)
	if err != nil {
		return bytes.Buffer{}, err
	}

	// Merge templates and inject
	var buf bytes.Buffer
	for _, tmpl := range c.Files {
		t := path.Join(renderOpts.ReleaseOptions.Name, tmpl.Name)
		if _, err := buf.WriteString(renderedTemplates[t]); err != nil {
			return bytes.Buffer{}, err
		}
	}

	return buf, nil
}

// Render returns a bytes buffer with the result of rendering a Helm chart
func (c *Chart) Render() (bytes.Buffer, error) {

	// Keep this slice synced with the contents of /charts/partials
	l5dPartials := []*chartutil.BufferedFile{
		{Name: "charts/partials/" + chartutil.ChartfileName},
		{Name: "charts/partials/templates/_proxy.tpl"},
		{Name: "charts/partials/templates/_proxy-init.tpl"},
		{Name: "charts/partials/templates/_volumes.tpl"},
		{Name: "charts/partials/templates/_resources.tpl"},
		{Name: "charts/partials/templates/_metadata.tpl"},
		{Name: "charts/partials/templates/_helpers.tpl"},
		{Name: "charts/partials/templates/_debug.tpl"},
		{Name: "charts/partials/templates/_capabilities.tpl"},
		{Name: "charts/partials/templates/_trace.tpl"},
		{Name: "charts/partials/templates/_nodeselector.tpl"},
		{Name: "charts/partials/templates/_tolerations.tpl"},
		{Name: "charts/partials/templates/_affinity.tpl"},
		{Name: "charts/partials/templates/_addons.tpl"},
		{Name: "charts/partials/templates/_validate.tpl"},
		{Name: "charts/partials/templates/_pull-secrets.tpl"},
	}
	return c.render(l5dPartials)
}

// RenderCNI returns a bytes buffer with the result of rendering a Helm chart
func (c *Chart) RenderCNI() (bytes.Buffer, error) {
	cniPartials := []*chartutil.BufferedFile{
		{Name: "charts/partials/" + chartutil.ChartfileName},
		{Name: "charts/partials/templates/_helpers.tpl"},
		{Name: "charts/partials/templates/_pull-secrets.tpl"},
	}
	return c.render(cniPartials)
}

// RenderNoPartials returns a bytes buffer with the result of rendering a Helm chart with no partials
func (c *Chart) RenderNoPartials() (bytes.Buffer, error) {
	return c.render([]*chartutil.BufferedFile{})
}

// ReadFile updates the buffered file with the data read from disk
func ReadFile(dir string, f *chartutil.BufferedFile) error {
	filename := dir + f.Name
	if dir == "" {
		filename = filename[7:]
	}
	file, err := static.Templates.Open(filename)
	if err != nil {
		return err
	}
	defer file.Close()

	buf := new(bytes.Buffer)
	if _, err := buf.ReadFrom(file); err != nil {
		return err
	}

	f.Data = buf.Bytes()
	return nil
}

// FilesReader reads all the files from a directory
func FilesReader(dir string, files []*chartutil.BufferedFile) error {
	for _, f := range files {
		if err := ReadFile(dir, f); err != nil {
			return err
		}
	}
	return nil
}

// InsertVersion returns the chart values file contents passed in
// with the version placeholder replaced with the current version
func InsertVersion(data []byte) []byte {
	dataWithVersion := strings.Replace(string(data), versionPlaceholder, version.Version, 1)
	return []byte(dataWithVersion)
}
