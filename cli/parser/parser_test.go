package parser

import (
	"testing"
)

func TestParseDeployFlagsValid(t *testing.T) {
	args := []string{"-c", "./app", "-n", "my-app", "-p", "9090"}
	cfg, apiURL, err := ParseDeployFlags(args, "http://localhost:8081")
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	if cfg.ContextPath != "./app" {
		t.Errorf("Expected context path './app', got '%s'", cfg.ContextPath)
	}
	if cfg.Name != "my-app" {
		t.Errorf("Expected name 'my-app', got '%s'", cfg.Name)
	}
	if cfg.Port != 9090 {
		t.Errorf("Expected port 9090, got %d", cfg.Port)
	}
	if apiURL != "http://localhost:8081" {
		t.Errorf("Expected default API URL, got '%s'", apiURL)
	}
}

func TestParseDeployFlagsMissingRequired(t *testing.T) {
	args := []string{"-n", "my-app"}
	_, _, err := ParseDeployFlags(args, "http://localhost:8081")
	if err == nil {
		t.Error("Expected error for missing context path flag, got nil")
	}
}

func TestParseDeregisterFlagsValid(t *testing.T) {
	args := []string{"--ip", "10.0.0.5", "--port", "8080"}
	ip, port, _, err := ParseDeregisterFlags(args, "http://localhost:8081")
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	if ip != "10.0.0.5" || port != 8080 {
		t.Errorf("Unexpected ip/port: %s:%d", ip, port)
	}
}

func TestParseWatchFlagsValid(t *testing.T) {
	args := []string{"--network", "custom-net"}
	net, _, err := ParseWatchFlags(args, "http://localhost:8081")
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	if net != "custom-net" {
		t.Errorf("Expected network 'custom-net', got '%s'", net)
	}
}
