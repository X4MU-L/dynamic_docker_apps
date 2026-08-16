package watcher

import (
	"encoding/json"

	"dynamic_docker_apps/cli/api_utils"
	"dynamic_docker_apps/cli/docker_utils"
	"dynamic_docker_apps/cli/domain"
	"dynamic_docker_apps/cli/logger"
)

func StartDockerWatcher(apiURL string, networkName string) error {
	logger.Info("Starting Docker container death listener on network '%s'...", networkName)

	return docker_utils.StreamContainerEvents(func(actorName, action string) {
		logger.Warn("Container '%s' event: %s. Checking upstreams for eviction...", actorName, action)
		evictContainerIfRegistered(apiURL, actorName)
	})
}

func evictContainerIfRegistered(apiURL string, actorName string) {
	data, err := api_utils.ListUpstreams(apiURL)
	if err != nil {
		logger.Error("Failed to fetch upstreams from Pingora API: %v", err)
		return
	}

	var items []domain.BackendItem
	if err := json.Unmarshal([]byte(data), &items); err != nil {
		return
	}

	for _, item := range items {
		if item.SNIName == actorName {
			if err := api_utils.DeregisterUpstream(apiURL, item.IP, item.Port); err == nil {
				logger.Success("Evicted dead container '%s' (%s:%d) from Pingora LB.", actorName, item.IP, item.Port)
			}
		}
	}
}
