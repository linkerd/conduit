package inject

import (
	"github.com/linkerd/linkerd2/pkg/k8s"
	v1 "k8s.io/api/core/v1"
)

// Uninject removes from the workload in conf the init and proxy containers,
// the TLS volumes and the extra annotations/labels that were added
func (conf *ResourceConfig) Uninject(report *Report) ([]byte, error) {
	if conf.podSpec == nil {
		return nil, nil
	}

	conf.uninjectPodSpec(report)
	conf.uninjectObjectMeta()
	return conf.YamlMarshalObj()
}

// Given a PodSpec, update the PodSpec in place with the sidecar
// and init-container uninjected
func (conf *ResourceConfig) uninjectPodSpec(report *Report) {
	t := conf.podSpec
	initContainers := []v1.Container{}
	for _, container := range t.InitContainers {
		if container.Name != k8s.InitContainerName {
			initContainers = append(initContainers, container)
		} else {
			report.Sidecar = true
		}
	}
	t.InitContainers = initContainers

	containers := []v1.Container{}
	for _, container := range t.Containers {
		if container.Name != k8s.ProxyContainerName {
			containers = append(containers, container)
		}
	}
	t.Containers = containers

	volumes := []v1.Volume{}
	for _, volume := range t.Volumes {
		// TODO: move those strings to constants
		if volume.Name != k8s.TLSTrustAnchorVolumeName && volume.Name != k8s.TLSSecretsVolumeName {
			volumes = append(volumes, volume)
		}
	}
	t.Volumes = volumes
}

func (conf *ResourceConfig) uninjectObjectMeta() {
	t := conf.objMeta
	newAnnotations := make(map[string]string)
	for key, val := range t.Annotations {
		if key != k8s.CreatedByAnnotation &&
			key != k8s.ProxyVersionAnnotation &&
			key != k8s.IdentityModeAnnotation {
			newAnnotations[key] = val
		}
	}
	t.Annotations = newAnnotations

	labels := make(map[string]string)
	for key, val := range t.Labels {
		keep := true
		for _, label := range k8s.InjectedLabels {
			if key == label {
				keep = false
				break
			}
		}
		if keep {
			labels[key] = val
		}
	}
	t.Labels = labels
}
