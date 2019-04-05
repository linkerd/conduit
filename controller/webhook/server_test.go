package webhook

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"reflect"
	"testing"

	"github.com/linkerd/linkerd2/controller/k8s"
)

func TestServe(t *testing.T) {
	t.Run("with empty http request body", func(t *testing.T) {
		k8sAPI, err := k8s.NewFakeAPI()
		if err != nil {
			panic(err)
		}
		testServer := &Server{nil, k8sAPI, nil, "linkerd"}

		in := bytes.NewReader(nil)
		request := httptest.NewRequest(http.MethodGet, "/", in)

		recorder := httptest.NewRecorder()
		testServer.serve(recorder, request)

		if recorder.Code != http.StatusOK {
			t.Errorf("HTTP response status mismatch. Expected: %d. Actual: %d", http.StatusOK, recorder.Code)
		}

		if reflect.DeepEqual(recorder.Body.Bytes(), []byte("")) {
			t.Errorf("Content mismatch. Expected HTTP response body to be empty %v", recorder.Body.Bytes())
		}
	})
}

func TestShutdown(t *testing.T) {
	server := &http.Server{Addr: ":0"}
	testServer := &Server{server, nil, nil, "linkerd"}

	go func() {
		if err := testServer.ListenAndServe(); err != nil {
			if err != http.ErrServerClosed {
				t.Errorf("Expected server to be gracefully shutdown with error: %q", http.ErrServerClosed)
			}
		}
	}()

	if err := testServer.Shutdown(); err != nil {
		t.Fatal("Unexpected error: ", err)
	}
}
