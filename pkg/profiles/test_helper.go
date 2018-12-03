package profiles

import (
	"fmt"
	"reflect"

	"github.com/ghodss/yaml"
	"github.com/linkerd/linkerd2/controller/gen/apis/serviceprofile/v1alpha1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func GenServiceProfile(service, namespace, controlPlaneNamespace string) v1alpha1.ServiceProfile {
	return v1alpha1.ServiceProfile{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "linkerd.io/v1alpha1",
			Kind:       "ServiceProfile",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      service + "." + namespace + ".svc.cluster.local",
			Namespace: controlPlaneNamespace,
		},
		Spec: v1alpha1.ServiceProfileSpec{
			Routes: []*v1alpha1.RouteSpec{
				&v1alpha1.RouteSpec{
					Name: "/authors/{id}",
					Condition: &v1alpha1.RequestMatch{
						Path:   "^/authors/\\d+$",
						Method: "POST",
					},
					ResponseClasses: []*v1alpha1.ResponseClass{
						&v1alpha1.ResponseClass{
							Condition: &v1alpha1.ResponseMatch{
								Status: &v1alpha1.Range{
									Min: 500,
									Max: 599,
								},
							},
							IsFailure: true,
						},
					},
				},
			},
		},
	}
}

func ServiceProfileYamlEquals(actual, expected v1alpha1.ServiceProfile) error {
	if !reflect.DeepEqual(actual, expected) {
		acutalYaml, err := yaml.Marshal(actual)
		if err != nil {
			return fmt.Errorf("Service profile mismatch but failed to marshal actual service profile: %v", err)
		}
		expectedYaml, err := yaml.Marshal(expected)
		if err != nil {
			return fmt.Errorf("Serivce profile mismatch but failed to marshal expected service profile: %v", err)
		}
		return fmt.Errorf("Expected [%s] but got [%s]", string(expectedYaml), string(acutalYaml))
	}
	return nil
}
