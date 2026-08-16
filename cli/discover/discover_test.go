package discover

import (
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"
	"time"
)

func TestProbeHealthEndpointSuccess(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/healthz" {
			w.WriteHeader(http.StatusOK)
			return
		}
		w.WriteHeader(http.StatusNotFound)
	}))
	defer server.Close()

	parts := strings.Split(server.Listener.Addr().String(), ":")
	ip := parts[0]
	port, _ := strconv.Atoi(parts[1])

	client := &http.Client{Timeout: 1 * time.Second}
	path, ok := probeHealthEndpoint(client, ip, port)
	if !ok || path != "/healthz" {
		t.Fatalf("expected /healthz path, got %s (ok=%v)", path, ok)
	}
}

func TestProbeHealthEndpointFailure(t *testing.T) {
	client := &http.Client{Timeout: 100 * time.Millisecond}
	_, ok := probeHealthEndpoint(client, "127.0.0.1", 59998)
	if ok {
		t.Fatalf("expected probe to fail on closed port")
	}
}
