package main

import (
	"flag"
	"fmt"
	"os"

	"dynamic_docker_apps/cli/api_utils"
	"dynamic_docker_apps/cli/deploy"
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
	fmt.Println("  deregister  Evict an upstream from Pingora by IP and port")
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
	ip, port, apiURL, err := parser.ParseDeregisterFlags(args, defaultApi)
	if err == flag.ErrHelp {
		os.Exit(0)
	}
	if err != nil {
		logger.Fatal("%v", err)
	}
	if err := api_utils.DeregisterUpstream(apiURL, ip, port); err != nil {
		logger.Fatal("Deregistration failed: %v", err)
	}
	logger.Success("Deregistered backend IP %s", ip)
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
