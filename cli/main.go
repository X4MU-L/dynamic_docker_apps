package main

import (
	"flag"
	"fmt"
	"os"
	"time"

	"dynamic_docker_apps/cli/api_utils"
	"dynamic_docker_apps/cli/deploy"
	"dynamic_docker_apps/cli/discover"
	"dynamic_docker_apps/cli/docker_utils"
	"dynamic_docker_apps/cli/logger"
	"dynamic_docker_apps/cli/parser"
	"dynamic_docker_apps/cli/watcher"
)

const defaultApiURL = "http://localhost:8081"

func main() {
	if len(os.Args) < 2 {
		printUsage()
		os.Exit(1)
	}

	cmd := os.Args[1]
	if cmd == "--help" || cmd == "-h" || cmd == "help" {
		printUsage()
		os.Exit(0)
	}

	apiURL := getApiURL()

	switch cmd {
	case "deploy":
		handleDeploy(os.Args[2:], apiURL)
	case "deregister":
		handleDeregister(os.Args[2:], apiURL)
	case "discover":
		handleDiscover(os.Args[2:], apiURL)
	case "list":
		handleList(os.Args[2:], apiURL)
	case "watch":
		handleWatch(os.Args[2:], apiURL)
	default:
		logger.Error("Unknown command: %s", cmd)
		printUsage()
		os.Exit(1)
	}
}

func getApiURL() string {
	url := os.Getenv("PINGORA_API_URL")
	if url == "" {
		return defaultApiURL
	}
	return url
}

func printUsage() {
	fmt.Println("Dynamic Pingora & Docker CLI Utility")
	fmt.Println("\nUsage:")
	fmt.Println("  deployer <command> [flags]")
	fmt.Println("\nCommands:")
	fmt.Println("  deploy      Build/pull and run container replicas on edge-net and register with Pingora")
	fmt.Println("  deregister  Evict an upstream from Pingora by container name or IP address")
	fmt.Println("  discover    Scan running containers, probe health endpoints, and register active backends")
	fmt.Println("  list        List all active Pingora upstreams")
	fmt.Println("  watch       Listen for Docker container death events and auto-evict endpoints")
	fmt.Println("\nFlags:")
	fmt.Println("  -h, --help  Show help menu")
}

func handleDeploy(args []string, defaultApi string) {
	cfg, apiURL, err := parser.ParseDeployFlags(args, defaultApi)
	if err == flag.ErrHelp {
		os.Exit(0)
	}
	if err != nil {
		logger.Fatal("%v", err)
	}
	if _, err := deploy.ExecuteDeployment(cfg, apiURL); err != nil {
		logger.Fatal("Deployment failed: %v", err)
	}
}

func handleDeregister(args []string, defaultApi string) {
	name, ip, port, stopContainer, drainTimeout, network, apiURL, err := parser.ParseDeregisterFlags(args, defaultApi)
	if err == flag.ErrHelp {
		os.Exit(0)
	}
	if err != nil {
		logger.Fatal("%v", err)
	}

	targetIP, targetName := resolveTargetInfo(name, ip, network)
	executeDeregistration(apiURL, targetIP, port, drainTimeout)

	if stopContainer {
		if drainTimeout > 0 {
			waitForBackendDrain(apiURL, targetIP, port)
		}
		stopTargetContainer(targetName, targetIP)
	}
}

func handleDiscover(args []string, defaultApi string) {
	apiURL, err := parser.ParseDiscoverFlags(args, defaultApi)
	if err == flag.ErrHelp {
		os.Exit(0)
	}
	if err != nil {
		logger.Fatal("%v", err)
	}
	if err := discover.RunCliDiscovery(apiURL); err != nil {
		logger.Fatal("Auto-discovery failed: %v", err)
	}
}

func executeDeregistration(apiURL, targetIP string, port, drainTimeout int) {
	if drainTimeout > 0 {
		if err := api_utils.DrainUpstream(apiURL, targetIP, port, "", drainTimeout); err != nil {
			logger.Fatal("Draining failed: %v", err)
		}
		logger.Info("Initiated graceful draining for backend IP %s (timeout: %ds)", targetIP, drainTimeout)
	} else {
		if err := api_utils.DeregisterUpstream(apiURL, targetIP, port); err != nil {
			logger.Fatal("Deregistration failed: %v", err)
		}
		logger.Success("Forcefully deregistered backend IP %s", targetIP)
	}
}

func waitForBackendDrain(apiURL, targetIP string, port int) {
	step := logger.StartStep("Waiting for Pingora LB to drain active requests for IP %s...", targetIP)
	for {
		st, err := api_utils.GetUpstreamStatus(apiURL, targetIP, port, "")
		if err != nil {
			step.FinishSuccess("Backend IP %s fully evicted from Pingora LB.", targetIP)
			break
		}
		rem := 0
		if st.RemainingDrainSecs != nil {
			rem = *st.RemainingDrainSecs
		}
		step.UpdateStream(fmt.Sprintf("Status: %s | Active Requests: %d | Remaining Drain: %ds", st.Status, st.ActiveRequests, rem))
		time.Sleep(1 * time.Second)
	}
}

func resolveTargetInfo(name, ip, network string) (string, string) {
	targetIP := ip
	targetName := name
	if targetIP == "" && targetName != "" {
		resolvedIP, err := docker_utils.ExtractContainerIP(targetName, network)
		if err != nil {
			logger.Fatal("Failed to resolve IP for container '%s': %v", targetName, err)
		}
		targetIP = resolvedIP
		logger.Info("Resolved container '%s' to IP %s", targetName, targetIP)
	}
	return targetIP, targetName
}

func stopTargetContainer(targetName, targetIP string) {
	containerToStop := targetName
	if containerToStop == "" {
		resolvedName, err := docker_utils.FindContainerNameByIP(targetIP)
		if err == nil {
			containerToStop = resolvedName
		}
	}
	if containerToStop != "" {
		logger.Info("Stopping and removing container '%s'...", containerToStop)
		docker_utils.StopAndRemoveContainer(containerToStop)
		logger.Success("Stopped and removed container '%s'.", containerToStop)
	} else {
		logger.Warn("Could not find running container for IP %s to stop.", targetIP)
	}
}

func handleList(args []string, defaultApi string) {
	for _, arg := range args {
		if arg == "-h" || arg == "--help" || arg == "help" {
			parser.PrintListHelp()
			os.Exit(0)
		}
	}
	output, err := api_utils.ListUpstreams(defaultApi)
	if err != nil {
		logger.Fatal("Failed to list upstreams: %v", err)
	}
	logger.Info("Active Pingora Upstreams:\n%s", output)
}

func handleWatch(args []string, defaultApi string) {
	network, apiURL, err := parser.ParseWatchFlags(args, defaultApi)
	if err == flag.ErrHelp {
		os.Exit(0)
	}
	if err != nil {
		logger.Fatal("%v", err)
	}
	if err := watcher.StartDockerWatcher(apiURL, network); err != nil {
		logger.Fatal("Watcher encountered error: %v", err)
	}
}
