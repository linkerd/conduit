package heartbeat

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"reflect"
	"testing"

	"github.com/linkerd/linkerd2/controller/api/public"
	"github.com/linkerd/linkerd2/pkg/k8s"
	"github.com/prometheus/common/model"
)

func TestK8sValues(t *testing.T) {
	testCases := []struct {
		namespace  string
		k8sConfigs []string
		expected   url.Values
	}{
		{
			"linkerd",
			[]string{`
apiVersion: v1
kind: Namespace
metadata:
  name: linkerd
  creationTimestamp: 2019-02-15T12:34:56Z`, `
kind: ConfigMap
apiVersion: v1
metadata:
  name: linkerd-config
  namespace: linkerd
data:
  install: |
    {"uuid":"fake-uuid"}`,
			},
			url.Values{
				"k8s-version":  []string{"v0.0.0-master+$Format:%h$"},
				"install-time": []string{"1550234096"},
				"uuid":         []string{"fake-uuid"},
			},
		},
		{
			"bad-ns",
			[]string{`
kind: ConfigMap
apiVersion: v1
metadata:
  name: linkerd-config
  namespace: linkerd
data:
  install: |
    {"uuid":"fake-uuid"}`,
			},
			url.Values{
				"k8s-version": []string{"v0.0.0-master+$Format:%h$"},
			},
		},
	}

	for i, tc := range testCases {
		tc := tc // pin
		t.Run(fmt.Sprintf("%d", i), func(t *testing.T) {
			k8sAPI, err := k8s.NewFakeAPI(tc.k8sConfigs...)
			if err != nil {
				t.Fatalf("NewFakeAPI returned an error: %s", err)
			}

			v := K8sValues(k8sAPI, tc.namespace)
			if !reflect.DeepEqual(v, tc.expected) {
				t.Fatalf("K8sValues returned: %+v, expected: %+v", v, tc.expected)
			}
		})
	}
}

func TestPromValues(t *testing.T) {
	testCases := []struct {
		promRes  model.Value
		expected url.Values
	}{
		{
			model.Vector{
				&model.Sample{
					Metric:    model.Metric{"pod": "emojivoto-meshed"},
					Value:     100.01,
					Timestamp: 456,
				},
			},
			url.Values{
				"rps":         []string{"100"},
				"meshed-pods": []string{"100"},
			},
		},
		{
			model.Vector{},
			url.Values{},
		},
	}

	for i, tc := range testCases {
		tc := tc // pin
		t.Run(fmt.Sprintf("%d", i), func(t *testing.T) {
			v := PromValues(&public.MockProm{Res: tc.promRes})
			if !reflect.DeepEqual(v, tc.expected) {
				t.Fatalf("PromValues returned: %+v, expected: %+v", v, tc.expected)
			}
		})
	}
}

func TestMergeValues(t *testing.T) {
	testCases := []struct {
		v1, v2, expected url.Values
	}{
		{
			url.Values{
				"a": []string{"b"},
				"c": []string{"d"},
			},
			url.Values{
				"e": []string{"f"},
				"g": []string{"h"},
			},
			url.Values{
				"a": []string{"b"},
				"c": []string{"d"},
				"e": []string{"f"},
				"g": []string{"h"},
			},
		},
		{
			url.Values{},
			url.Values{},
			url.Values{},
		},
	}

	for i, tc := range testCases {
		tc := tc // pin
		t.Run(fmt.Sprintf("%d", i), func(t *testing.T) {
			v := MergeValues(tc.v1, tc.v2)
			if !reflect.DeepEqual(v, tc.expected) {
				t.Fatalf("MergeValues returned: %+v, expected: %+v", v, tc.expected)
			}
		})
	}
}

func TestSend(t *testing.T) {
	testCases := []struct {
		v   url.Values
		err error
	}{
		{
			url.Values{
				"a": []string{"b"},
				"c": []string{"d"},
			},
			nil,
		},
	}

	for i, tc := range testCases {
		tc := tc // pin
		t.Run(fmt.Sprintf("%d", i), func(t *testing.T) {
			ts := httptest.NewServer(http.HandlerFunc(
				func(w http.ResponseWriter, r *http.Request) {
					if !reflect.DeepEqual(r.URL.Query(), tc.v) {
						t.Fatalf("Send queried for: %+v, expected: %+v", r.URL.Query(), tc.v)
					}
					w.Write([]byte(`{"stable":"stable-a.b.c","edge":"edge-d.e.f"}`))
				}),
			)
			defer ts.Close()

			err := send(ts.Client(), ts.URL, tc.v)
			if !reflect.DeepEqual(err, tc.err) {
				t.Fatalf("Send returned: %+v, expected: %+v", err, tc.err)
			}
		})
	}
}
