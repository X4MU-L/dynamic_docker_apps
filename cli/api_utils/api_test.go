package api_utils

import (
	"fmt"
	"strings"
	"testing"

	"dynamic_docker_apps/cli/domain"
)

type MockCommandRunner struct {
	MockFunc func(name string, args ...string) ([]byte, error)
}

func (m MockCommandRunner) RunCommand(name string, args ...string) ([]byte, error) {
	return m.MockFunc(name, args...)
}

func TestCheckApiServerHealthSuccess(t *testing.T) {
	mock := MockCommandRunner{
		MockFunc: func(name string, args ...string) ([]byte, error) {
			return []byte(""), nil
		},
	}
	SetCommandRunner(mock)
	defer SetCommandRunner(RealCommandRunner{})

	if err := CheckApiServerHealth("http://pingora-lb:8081"); err != nil {
		t.Errorf("Expected health check success, got: %v", err)
	}
}

func TestCheckApiServerHealthContainerNotRunning(t *testing.T) {
	mock := MockCommandRunner{
		MockFunc: func(name string, args ...string) ([]byte, error) {
			return []byte("Error response from daemon: No such container: pingora-lb"), fmt.Errorf("exit status 1")
		},
	}
	SetCommandRunner(mock)
	defer SetCommandRunner(RealCommandRunner{})

	err := CheckApiServerHealth("http://pingora-lb:8081")
	if err == nil {
		t.Fatal("Expected error when container is not running, got nil")
	}
	if !strings.Contains(err.Error(), "pingora-lb") {
		t.Errorf("Unexpected error message: %v", err)
	}
}

func TestRegisterUpstreamSuccess(t *testing.T) {
	mock := MockCommandRunner{
		MockFunc: func(name string, args ...string) ([]byte, error) {
			return []byte(`{"status":"registered"}`), nil
		},
	}
	SetCommandRunner(mock)
	defer SetCommandRunner(RealCommandRunner{})

	target := domain.NewUpstreamTarget("127.0.0.1", 8080, "test", "/health")
	if err := RegisterUpstream("http://pingora-lb:8081", target); err != nil {
		t.Errorf("Expected register success, got: %v", err)
	}
}

func TestRegisterUpstreamConflictError(t *testing.T) {
	mock := MockCommandRunner{
		MockFunc: func(name string, args ...string) ([]byte, error) {
			cmdStr := strings.Join(args, " ")
			if strings.Contains(cmdStr, "/upstreams") {
				return []byte(`HTTP_ERROR:409:{"error":"Backend already registered","code":409}`), fmt.Errorf("exit status 1")
			}
			return []byte(""), nil
		},
	}
	SetCommandRunner(mock)
	defer SetCommandRunner(RealCommandRunner{})

	target := domain.NewUpstreamTarget("127.0.0.1", 8080, "test", "/health")
	err := RegisterUpstream("http://pingora-lb:8081", target)
	if err == nil {
		t.Fatal("Expected error on duplicate registration, got nil")
	}
	if err.Error() != "API error (409): Backend already registered" {
		t.Errorf("Unexpected error message: %v", err)
	}
}

func TestListUpstreamsSuccess(t *testing.T) {
	mock := MockCommandRunner{
		MockFunc: func(name string, args ...string) ([]byte, error) {
			cmdStr := strings.Join(args, " ")
			if strings.Contains(cmdStr, "/upstreams") {
				return []byte(`[{"ip":"127.0.0.1","port":8080}]`), nil
			}
			return []byte(""), nil
		},
	}
	SetCommandRunner(mock)
	defer SetCommandRunner(RealCommandRunner{})

	list, err := ListUpstreams("http://pingora-lb:8081")
	if err != nil {
		t.Fatalf("Unexpected error listing upstreams: %v", err)
	}
	if list != `[{"ip":"127.0.0.1","port":8080}]` {
		t.Errorf("Unexpected list response: %s", list)
	}
}
