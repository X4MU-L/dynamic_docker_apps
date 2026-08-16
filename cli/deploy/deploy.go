package deploy

import (
	"fmt"
	"strings"

	"dynamic_docker_apps/cli/api_utils"
	"dynamic_docker_apps/cli/docker_utils"
	"dynamic_docker_apps/cli/domain"
	"dynamic_docker_apps/cli/health"
	"dynamic_docker_apps/cli/logger"
)

func ExecuteDeployment(config domain.DeploymentConfig, apiURL string) (string, error) {
	if err := api_utils.CheckApiServerHealth(apiURL); err != nil {
		logger.Error("%v", err)
		return "", err
	}

	containerName := strings.ToLower(config.Name)
	if containerName == "" {
		containerName = domain.GenerateContainerName("app")
	}
	if docker_utils.ContainerExists(containerName) {
		return "", fmt.Errorf("container '%s' is already running in Docker", containerName)
	}

	domainSuffix := strings.ToLower(config.DomainSuffix)
	if domainSuffix == "" {
		domainSuffix = domain.DefaultDomainSuffix
	}
	hostname := fmt.Sprintf("%s.%s", containerName, domainSuffix)

	imageTag, err := prepareImage(config, containerName)
	if err != nil {
		return "", err
	}

	if err := docker_utils.RunContainer(imageTag, containerName, hostname, config.Network); err != nil {
		return "", err
	}

	return registerAndCompleteDeployment(config, apiURL, containerName)
}

func prepareImage(config domain.DeploymentConfig, containerName string) (string, error) {
	if config.Image != "" {
		if err := docker_utils.EnsureImageAvailable(config.Image, config.Username, config.Password); err != nil {
			return "", err
		}
		return config.Image, nil
	}

	imageTag := fmt.Sprintf("%s:latest", containerName)
	if err := docker_utils.BuildImage(config.ContextPath, imageTag); err != nil {
		return "", err
	}
	return imageTag, nil
}

func registerAndCompleteDeployment(config domain.DeploymentConfig, apiURL, containerName string) (string, error) {
	ipAddress, err := docker_utils.ExtractContainerIP(containerName, config.Network)
	if err != nil {
		docker_utils.StopAndRemoveContainer(containerName)
		return "", err
	}
	logger.Info("Assigned container IP: %s", ipAddress)

	stepHealth := logger.StartStep("Probing readiness health for %s:%d%s...", ipAddress, config.Port, config.HealthEndpoint)
	if !health.WaitForReadiness(config.Network, ipAddress, config.Port, config.HealthEndpoint, config.TimeoutSecs) {
		stepHealth.FinishError("Readiness health probe failed for %s", containerName)
		docker_utils.StopAndRemoveContainer(containerName)
		return "", fmt.Errorf("container %s failed health probe", containerName)
	}
	stepHealth.FinishSuccess("Container %s is healthy.", containerName)

	target := domain.NewUpstreamTarget(ipAddress, config.Port, containerName, config.DomainSuffix, config.HealthEndpoint)
	stepReg := logger.StartStep("Registering %s (Hostname: %s) with Pingora LB...", containerName, target.SNIName)
	if err := api_utils.RegisterUpstream(apiURL, target); err != nil {
		stepReg.FinishError("Pingora LB registration failed for %s", containerName)
		docker_utils.StopAndRemoveContainer(containerName)
		return "", err
	}
	stepReg.FinishSuccess("Registered '%s' (Hostname: %s) with Pingora LB.", containerName, target.SNIName)

	logger.Success("Deployment complete: '%s' (Hostname: %s) is active and routing.", containerName, target.SNIName)
	return containerName, nil
}
