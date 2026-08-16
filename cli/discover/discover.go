package discover

import (
	"fmt"
	"net/http"
	"os/exec"
	"strings"
	"time"

	"dynamic_docker_apps/cli/api_utils"
	"dynamic_docker_apps/cli/docker_utils"
	"dynamic_docker_apps/cli/domain"
	"dynamic_docker_apps/cli/logger"
)

var HealthProbePaths = []string{"/health", "/healthz", "/api/health", "/api/healthz"}

type DiscoveredApp struct {
	Name     string
	Hostname string
	IP       string
	Port     int
}

func RunCliDiscovery(apiURL string) error {
	logger.Info("🔍 Starting CLI Auto-Discovery of running container backends...")
	if err := api_utils.CheckApiServerHealth(apiURL); err != nil {
		return fmt.Errorf("pingora control API unreachable at %s: %w", apiURL, err)
	}

	containers, err := listRunningContainers()
	if err != nil {
		return fmt.Errorf("failed to list containers: %w", err)
	}

	count := processContainers(containers, apiURL)
	logger.Success("Auto-Discovery complete. Registered %d active backend(s).", count)
	return nil
}

func listRunningContainers() ([]string, error) {
	cmd := exec.Command("docker", "ps", "--format", "{{.Names}}")
	out, err := cmd.Output()
	if err != nil {
		return nil, err
	}
	lines := strings.Split(strings.TrimSpace(string(out)), "\n")
	var names []string
	for _, l := range lines {
		trimmed := strings.TrimSpace(l)
		if trimmed != "" && (!strings.Contains(trimmed, "pingora-lb") && !strings.Contains(trimmed, "pingora-discover")) {
			names = append(names, trimmed)
		}
	}
	return names, nil
}

func processContainers(containers []string, apiURL string) int {
	registered := 0
	client := &http.Client{Timeout: 2 * time.Second}
	for _, name := range containers {
		target, err := inspectApp(name)
		if err != nil {
			logger.Info("⏭️ Skipping '%s': %v", name, err)
			continue
		}
		if ep, ok := probeHealthEndpoint(client, target.IP, target.Port); ok {
			if registerApp(apiURL, target, ep) == nil {
				registered++
			}
		} else {
			logger.Info("⏭️ Skipping '%s' (%s:%d): no health endpoint reachable", name, target.IP, target.Port)
		}
	}
	return registered
}

func inspectApp(name string) (*DiscoveredApp, error) {
	ip, err := docker_utils.ExtractContainerIP(name, "edge-net")
	if err != nil || ip == "" {
		return nil, fmt.Errorf("failed to extract IP")
	}
	hostname := fmt.Sprintf("%s.edge.local", strings.ToLower(name))
	return &DiscoveredApp{
		Name:     name,
		Hostname: hostname,
		IP:       ip,
		Port:     8080,
	}, nil
}

func probeHealthEndpoint(client *http.Client, ip string, port int) (string, bool) {
	for _, path := range HealthProbePaths {
		url := fmt.Sprintf("http://%s:%d%s", ip, port, path)
		if ip == "127.0.0.1" || ip == "localhost" {
			if probeViaHostClient(client, url) {
				return path, true
			}
			continue
		}
		if probeViaContainerExec(url) {
			return path, true
		}
	}
	return "", false
}

func probeViaHostClient(client *http.Client, url string) bool {
	resp, err := client.Get(url)
	if err == nil && resp.StatusCode == http.StatusOK {
		_ = resp.Body.Close()
		return true
	}
	if resp != nil {
		_ = resp.Body.Close()
	}
	return false
}

func probeViaContainerExec(url string) bool {
	pyCmd := fmt.Sprintf("import urllib.request, sys; resp = urllib.request.urlopen('%s'); sys.exit(0 if resp.status == 200 else 1)", url)
	cmd := exec.Command("docker", "exec", "pingora-lb", "python3", "-c", pyCmd)
	return cmd.Run() == nil
}

func registerApp(apiURL string, target *DiscoveredApp, healthEP string) error {
	payload := domain.UpstreamTarget{
		IP:             target.IP,
		Port:           target.Port,
		SNIName:        target.Hostname,
		HealthEndpoint: healthEP,
	}
	if err := api_utils.RegisterUpstream(apiURL, payload); err != nil {
		return err
	}
	logger.Success("Registered '%s' (%s:%d, health: %s)", target.Hostname, target.IP, target.Port, healthEP)
	return nil
}
