package parser

import (
	"testing"
)

func TestParseDeployFlagsValidContext(t *testing.T) {
	args := []string{"-c", "./app", "-n", "my-app-1", "-p", "9090"}
	cfg, apiURL, err := ParseDeployFlags(args, "http://localhost:8081")
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	if cfg.ContextPath != "./app" || cfg.Name != "my-app-1" || cfg.Port != 9090 {
		t.Errorf("Unexpected deployment config values: %+v", cfg)
	}
	if apiURL != "http://localhost:8081" {
		t.Errorf("Expected default API URL, got '%s'", apiURL)
	}
}

func TestParseDeployFlagsValidImage(t *testing.T) {
	args := []string{"-i", "nginx:latest", "-u", "user", "--password", "pass", "-n", "my-nginx"}
	cfg, _, err := ParseDeployFlags(args, "http://localhost:8081")
	if err != nil {
		t.Fatalf("Unexpected error parsing image flags: %v", err)
	}
	if cfg.Image != "nginx:latest" || cfg.Username != "user" || cfg.Password != "pass" || cfg.Name != "my-nginx" {
		t.Errorf("Unexpected image deployment config values: %+v", cfg)
	}
}

func TestParseDeployFlagsMissingImageAndContext(t *testing.T) {
	args := []string{"-n", "my-app"}
	_, _, err := ParseDeployFlags(args, "http://localhost:8081")
	if err == nil {
		t.Error("Expected error when both image and context path are missing, got nil")
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
