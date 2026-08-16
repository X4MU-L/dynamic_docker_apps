package parser

import (
	"testing"
)

func TestParseDeployFlagsValid(t *testing.T) {
	validNames := []string{"my-app-1", "APP-1", "App-1", "my-APP-10"}
	for _, name := range validNames {
		args := []string{"-c", "./app", "-n", name, "-p", "9090"}
		cfg, apiURL, err := ParseDeployFlags(args, "http://localhost:8081")
		if err != nil {
			t.Fatalf("Expected valid name '%s', got error: %v", name, err)
		}
		if cfg.Name != name {
			t.Errorf("Expected name '%s', got '%s'", name, cfg.Name)
		}
		if apiURL != "http://localhost:8081" {
			t.Errorf("Expected default API URL, got '%s'", apiURL)
		}
	}
}

func TestParseDeployFlagsInvalidName(t *testing.T) {
	invalidNames := []string{"My_App", "app@1", "-app", "app-", "app name"}
	for _, invalid := range invalidNames {
		args := []string{"-c", "./app", "-n", invalid}
		_, _, err := ParseDeployFlags(args, "http://localhost:8081")
		if err == nil {
			t.Errorf("Expected error for invalid container name '%s', got nil", invalid)
		}
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
