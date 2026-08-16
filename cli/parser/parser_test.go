package parser

import (
	"flag"
	"testing"
)

func TestParseDeployFlagsValidContext(t *testing.T) {
	args := []string{"-c", "./app", "-n", "my-app-1", "-r", "3", "-p", "9090"}
	cfg, apiURL, err := ParseDeployFlags(args, "http://localhost:8081")
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	if cfg.ContextPath != "./app" || cfg.Name != "my-app-1" || cfg.Replicas != 3 || cfg.Port != 9090 {
		t.Errorf("Unexpected deployment config values: %+v", cfg)
	}
	if apiURL != "http://localhost:8081" {
		t.Errorf("Expected default API URL, got '%s'", apiURL)
	}
}

func TestParseDeployFlagsHelp(t *testing.T) {
	args := []string{"--help"}
	_, _, err := ParseDeployFlags(args, "http://localhost:8081")
	if err != flag.ErrHelp {
		t.Errorf("Expected flag.ErrHelp for --help, got %v", err)
	}
}

func TestParseDeployFlagsInvalidReplicas(t *testing.T) {
	args := []string{"-c", "./app", "-r", "0"}
	_, _, err := ParseDeployFlags(args, "http://localhost:8081")
	if err == nil {
		t.Error("Expected error for 0 replicas, got nil")
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

func TestParseDeregisterFlagsValidIP(t *testing.T) {
	args := []string{"--ip", "10.0.0.5", "--port", "8080"}
	_, ip, port, stop, _, _, err := ParseDeregisterFlags(args, "http://localhost:8081")
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	if ip != "10.0.0.5" || port != 8080 || stop != false {
		t.Errorf("Unexpected ip/port/stop: %s:%d stop=%v", ip, port, stop)
	}
}

func TestParseDeregisterFlagsValidNameAndStop(t *testing.T) {
	args := []string{"-n", "my-container", "-s"}
	name, ip, _, stop, _, _, err := ParseDeregisterFlags(args, "http://localhost:8081")
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	if name != "my-container" || ip != "" || stop != true {
		t.Errorf("Expected name 'my-container' stop=true, got name='%s' stop=%v", name, stop)
	}
}

func TestParseDeregisterFlagsMissingNameAndIP(t *testing.T) {
	args := []string{"--port", "8080"}
	_, _, _, _, _, _, err := ParseDeregisterFlags(args, "http://localhost:8081")
	if err == nil {
		t.Error("Expected error when both name and ip are missing, got nil")
	}
}

func TestParseDeregisterFlagsHelp(t *testing.T) {
	args := []string{"-h"}
	_, _, _, _, _, _, err := ParseDeregisterFlags(args, "http://localhost:8081")
	if err != flag.ErrHelp {
		t.Errorf("Expected flag.ErrHelp for -h, got %v", err)
	}
}

func TestParseWatchFlagsHelp(t *testing.T) {
	args := []string{"help"}
	_, _, err := ParseWatchFlags(args, "http://localhost:8081")
	if err != flag.ErrHelp {
		t.Errorf("Expected flag.ErrHelp for help, got %v", err)
	}
}
