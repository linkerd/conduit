package labels

import (
	"strconv"

	corev1 "k8s.io/api/core/v1"
)

const (
	// VizAnnotationsPrefix is the prefix of all viz-related annotations
	VizAnnotationsPrefix = "viz.linkerd.io"

	// VizTapEnabled is set by the tap-injector component when tap has been
	// enabled on a pod.
	VizTapEnabled = VizAnnotationsPrefix + "/tap-enabled"
)

// IsTapEnabled returns true if a pod has an annotation indicating that tap
// is enabled.
func IsTapEnabled(pod *corev1.Pod) bool {
	valStr := pod.GetAnnotations()[VizTapEnabled]
	if valStr != "" {
		valBool, err := strconv.ParseBool(valStr)
		if err == nil && valBool {
			return true
		}
	}
	return false
}
