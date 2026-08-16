package health

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"strconv"
	"testing"
)

func TestWaitForReadinessSuccess(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/health" {
			w.WriteHeader(http.StatusOK)
		}
	}))
	defer ts.Close()

	u, _ := url.Parse(ts.URL)
	port, _ := strconv.Atoi(u.Port())

	ok := WaitForReadiness("edge-net", u.Hostname(), port, "/health", 5)
	if !ok {
		t.Error("Expected readiness check to succeed, got false")
	}
}

func TestWaitForReadinessTimeout(t *testing.T) {
	ok := WaitForReadiness("edge-net", "127.0.0.1", 59999, "/health", 1)
	if ok {
		t.Error("Expected readiness check to fail on unreachable port, got true")
	}
}
