package api_utils

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"dynamic_docker_apps/cli/domain"
)

func createMockServer(handler http.HandlerFunc) *httptest.Server {
	return httptest.NewServer(handler)
}

func TestCheckApiServerHealthSuccess(t *testing.T) {
	ts := createMockServer(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/health" {
			w.WriteHeader(http.StatusOK)
		}
	})
	defer ts.Close()

	if err := CheckApiServerHealth(ts.URL); err != nil {
		t.Errorf("Expected health check success, got: %v", err)
	}
}

func TestRegisterUpstreamSuccess(t *testing.T) {
	ts := createMockServer(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/health" || r.URL.Path == "/upstreams" {
			w.WriteHeader(http.StatusOK)
		}
	})
	defer ts.Close()

	target := domain.NewUpstreamTarget("127.0.0.1", 8080, "test", "/health")
	if err := RegisterUpstream(ts.URL, target); err != nil {
		t.Errorf("Expected register success, got: %v", err)
	}
}

func TestRegisterUpstreamErrorResponse(t *testing.T) {
	ts := createMockServer(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/health" {
			w.WriteHeader(http.StatusOK)
			return
		}
		w.WriteHeader(http.StatusConflict)
		_, _ = fmt.Fprint(w, `{"error":"Backend already registered","code":409}`)
	})
	defer ts.Close()

	target := domain.NewUpstreamTarget("127.0.0.1", 8080, "test", "/health")
	err := RegisterUpstream(ts.URL, target)
	if err == nil {
		t.Fatal("Expected error on duplicate registration, got nil")
	}
	if err.Error() != "API error (409): Backend already registered" {
		t.Errorf("Unexpected error message: %v", err)
	}
}

func TestListUpstreamsSuccess(t *testing.T) {
	ts := createMockServer(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/health" {
			w.WriteHeader(http.StatusOK)
		} else if r.URL.Path == "/upstreams" {
			w.WriteHeader(http.StatusOK)
			_, _ = fmt.Fprint(w, `[{"ip":"127.0.0.1","port":8080}]`)
		}
	})
	defer ts.Close()

	list, err := ListUpstreams(ts.URL)
	if err != nil {
		t.Fatalf("Unexpected error listing upstreams: %v", err)
	}
	if list != `[{"ip":"127.0.0.1","port":8080}]` {
		t.Errorf("Unexpected list response: %s", list)
	}
}
