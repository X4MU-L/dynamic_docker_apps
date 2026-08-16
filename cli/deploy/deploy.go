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

	baseName := strings.ToLower(config.Name)
	if baseName == "" {
		baseName = domain.GenerateContainerName("app")
	}

	imageTag, err := prepareImage(config, baseName)
	if err != nil {
		return "", err
	}

	return deployReplicas(config, apiURL, imageTag, baseName)
}

func deployReplicas(config domain.DeploymentConfig, apiURL, imageTag, baseName string) (string, error) {
	replicas := config.Replicas
	if replicas < 1 {
		replicas = 1
	}

	var deployed []string
	for i := 1; i <= replicas; i++ {
		instanceName := baseName
		if replicas > 1 {
			instanceName = fmt.Sprintf("%s-%d", baseName, i)
		}

		if err := deploySingleInstance(config, apiURL, imageTag, instanceName); err != nil {
			logger.Error("Failed to deploy replica '%s': %v", instanceName, err)
			return "", err
		}
		deployed = append(deployed, instanceName)
	}

	logger.Success("Deployment complete: %d replica(s) [%s] active and routing.", len(deployed), strings.Join(deployed, ", "))
	return baseName, nil
}

func deploySingleInstance(config domain.DeploymentConfig, apiURL, imageTag, instanceName string) error {
	if docker_utils.ContainerExists(instanceName) {
		return fmt.Errorf("container '%s' is already running in Docker", instanceName)
	}

	domainSuffix := strings.ToLower(config.DomainSuffix)
	if domainSuffix == "" {
		domainSuffix = domain.DefaultDomainSuffix
	}
	hostname := fmt.Sprintf("%s.%s", instanceName, domainSuffix)

	if err := docker_utils.RunContainer(imageTag, instanceName, hostname, config.Network); err != nil {
		return err
	}

	return registerAndCompleteDeployment(config, apiURL, instanceName)
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

func registerAndCompleteDeployment(config domain.DeploymentConfig, apiURL, instanceName string) error {
	ipAddress, err := docker_utils.ExtractContainerIP(instanceName, config.Network)
	if err != nil {
		docker_utils.StopAndRemoveContainer(instanceName)
		return err
	}

	stepHealth := logger.StartStep("Probing readiness health for %s (%s:%d)...", instanceName, ipAddress, config.Port)
	if !health.WaitForReadiness(config.Network, ipAddress, config.Port, config.HealthEndpoint, config.TimeoutSecs) {
		stepHealth.FinishError("Readiness health probe failed for %s", instanceName)
		docker_utils.StopAndRemoveContainer(instanceName)
		return fmt.Errorf("container %s failed health probe", instanceName)
	}
	stepHealth.FinishSuccess("Container %s is healthy.", instanceName)

	target := domain.NewUpstreamTarget(ipAddress, config.Port, instanceName, config.DomainSuffix, config.HealthEndpoint)
	stepReg := logger.StartStep("Registering %s (Hostname: %s) with Pingora LB...", instanceName, target.SNIName)
	if err := api_utils.RegisterUpstream(apiURL, target); err != nil {
		stepReg.FinishError("Pingora LB registration failed for %s", instanceName)
		docker_utils.StopAndRemoveContainer(instanceName)
		return err
	}
	stepReg.FinishSuccess("Registered '%s' (Hostname: %s) with Pingora LB.", instanceName, target.SNIName)
	return nil
}
