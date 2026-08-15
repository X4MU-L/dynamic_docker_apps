package docker_utils

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os/exec"
	"strings"

	"dynamic_docker_apps/cli/logger"
)

func EnsureNetworkExists(networkName string) error {
	inspectCmd := exec.Command("docker", "network", "inspect", networkName)
	if err := inspectCmd.Run(); err == nil {
		return nil
	}

	logger.Info("Network '%s' not found. Creating Docker bridge network...", networkName)
	args := []string{
		"network", "create",
		"--subnet", "172.30.0.0/16",
		"--label", fmt.Sprintf("com.docker.compose.network=%s", networkName),
		networkName,
	}
	createCmd := exec.Command("docker", args...)
	out, err := createCmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("failed to create docker network '%s': %s (%v)", networkName, string(out), err)
	}
	logger.Success("Docker network '%s' created.", networkName)
	return nil
}

func BuildImage(contextPath, tag string) error {
	step := logger.StartStep("Building Docker image from '%s'...", contextPath)
	cmd := exec.Command("docker", "build", "-t", tag, contextPath)
	if err := runBufferedStepCmd(cmd, step); err != nil {
		step.FinishError("Docker build failed for %s", tag)
		return err
	}
	step.FinishSuccess("Docker image '%s' built successfully.", tag)
	return nil
}

func RunContainer(tag, containerName, networkName string) error {
	if err := EnsureNetworkExists(networkName); err != nil {
		return err
	}

	step := logger.StartStep("Running container '%s' on network '%s'...", containerName, networkName)
	args := []string{"run", "-d", "--name", containerName, "--network", networkName, tag}
	cmd := exec.Command("docker", args...)
	if err := runBufferedStepCmd(cmd, step); err != nil {
		step.FinishError("Docker run failed for %s", containerName)
		return err
	}
	step.FinishSuccess("Container '%s' started on network '%s'.", containerName, networkName)
	return nil
}

func runBufferedStepCmd(cmd *exec.Cmd, step *logger.LiveStep) error {
	pipe, err := cmd.StdoutPipe()
	if err != nil {
		return err
	}
	cmd.Stderr = cmd.Stdout

	if err := cmd.Start(); err != nil {
		return err
	}

	scanner := bufio.NewScanner(pipe)
	for scanner.Scan() {
		step.UpdateStream(scanner.Text())
	}
	_ = scanner.Err()

	return cmd.Wait()
}

func ExtractContainerIP(containerName, networkName string) (string, error) {
	const format = "{{range .NetworkSettings.Networks}}{{.IPAddress}}{{end}}"
	cmd := exec.Command("docker", "inspect", "-f", format, containerName)
	out, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("docker inspect failed: %v", err)
	}
	ip := strings.TrimSpace(string(out))
	if ip == "" {
		return "", fmt.Errorf("no IP address found for container %s on network %s", containerName, networkName)
	}
	return ip, nil
}

func StopAndRemoveContainer(containerName string) {
	cmd := exec.Command("docker", "rm", "-f", containerName)
	_ = cmd.Run()
}

func StreamContainerEvents(handler func(actorName, action string)) error {
	cmd := exec.Command("docker", "events", "--filter", "type=container", "--format", "{{json .}}")
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return err
	}
	if err := cmd.Start(); err != nil {
		return err
	}

	scanner := bufio.NewScanner(stdout)
	for scanner.Scan() {
		parseAndHandleEvent(scanner.Bytes(), handler)
	}
	if err := scanner.Err(); err != nil {
		return err
	}
	return cmd.Wait()
}

func parseAndHandleEvent(data []byte, handler func(actorName, action string)) {
	var evt struct {
		Action string `json:"Action"`
		Actor  struct {
			Attributes struct {
				Name string `json:"name"`
			} `json:"Attributes"`
		} `json:"Actor"`
	}
	if err := json.Unmarshal(data, &evt); err == nil {
		if evt.Action == "die" || evt.Action == "stop" || evt.Action == "destroy" {
			handler(evt.Actor.Attributes.Name, evt.Action)
		}
	}
}
