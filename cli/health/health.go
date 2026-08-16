package health

import (
	"fmt"
	"os/exec"
	"strings"
	"time"
)

func WaitForReadiness(networkName, ip string, port int, endpoint string, timeoutSecs int) bool {
	healthURL := fmt.Sprintf("http://%s:%d/%s", ip, port, strings.TrimLeft(endpoint, "/"))
	deadline := time.Now().Add(time.Duration(timeoutSecs) * time.Second)

	for time.Now().Before(deadline) {
		// if probeDirectHost(healthURL) {
		// 	return true
		// }
		if probeDockerExec(healthURL) {
			return true
		}
		if probeDockerNetworkRun(networkName, healthURL) {
			return true
		}
		time.Sleep(1 * time.Second)
	}

	return false
}

// func probeDirectHost(healthURL string) bool {
// 	client := &http.Client{Timeout: 1 * time.Second}
// 	resp, err := client.Get(healthURL)
// 	if err == nil && resp.StatusCode == http.StatusOK {
// 		resp.Body.Close()
// 		return true
// 	}
// 	if resp != nil {
// 		resp.Body.Close()
// 	}
// 	return false
// }

func probeDockerExec(healthURL string) bool {
	pyCmd := fmt.Sprintf("import urllib.request; resp = urllib.request.urlopen('%s'); exit(0 if resp.status == 200 else 1)", healthURL)
	cmd := exec.Command("docker", "exec", "pingora-lb", "python3", "-c", pyCmd)
	return cmd.Run() == nil
}

func probeDockerNetworkRun(networkName, healthURL string) bool {
	pyCmd := fmt.Sprintf("import urllib.request; resp = urllib.request.urlopen('%s'); exit(0 if resp.status == 200 else 1)", healthURL)
	cmd := exec.Command("docker", "run", "--rm", "--network", networkName, "python:3.11-slim", "python3", "-c", pyCmd)
	return cmd.Run() == nil
}
