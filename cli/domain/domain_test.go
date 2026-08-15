package domain

import (
	"strings"
	"testing"
)

func TestGenerateContainerName(t *testing.T) {
	name := GenerateContainerName("testapp")
	if !strings.HasPrefix(name, "testapp-") {
		t.Errorf("Expected prefix 'testapp-', got: %s", name)
	}
	if len(name) < 14 {
		t.Errorf("Unexpected length for container name: %d", len(name))
	}
}

func TestNewUpstreamTargetDefaults(t *testing.T) {
	target := NewUpstreamTarget("192.168.1.10", 8080, "", "")
	if target.SNIName != "192.168.1.10" {
		t.Errorf("Expected SNI fallback to IP, got: %s", target.SNIName)
	}
	if target.HealthEndpoint != "/health" {
		t.Errorf("Expected health endpoint fallback to '/health', got: %s", target.HealthEndpoint)
	}
}

func TestNewUpstreamTargetExplicit(t *testing.T) {
	target := NewUpstreamTarget("10.0.0.1", 9000, "custom-sni", "/custom-health")
	if target.SNIName != "custom-sni" {
		t.Errorf("Expected SNI 'custom-sni', got: %s", target.SNIName)
	}
	if target.HealthEndpoint != "/custom-health" {
		t.Errorf("Expected health endpoint '/custom-health', got: %s", target.HealthEndpoint)
	}
}
