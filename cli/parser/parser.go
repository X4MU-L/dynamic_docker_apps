package parser

import (
	"flag"
	"fmt"

	"dynamic_docker_apps/cli/domain"
)

func ParseDeployFlags(args []string, defaultAPI string) (domain.DeploymentConfig, string, error) {
	fs := flag.NewFlagSet("deploy", flag.ContinueOnError)
	contextPath := fs.String("context", "", "Path to Docker context build directory")
	fs.StringVar(contextPath, "c", "", "Path to Docker context build directory (shorthand)")
	name := fs.String("name", "", "Container instance name")
	fs.StringVar(name, "n", "", "Container instance name (shorthand)")
	port := fs.Int("port", 8080, "Container app port")
	fs.IntVar(port, "p", 8080, "Container app port (shorthand)")
	network := fs.String("network", "edge-net", "Docker bridge network")
	healthEp := fs.String("health-endpoint", "/health", "Health probe path")
	timeout := fs.Int("timeout", 30, "Health probe timeout seconds")
	apiURL := fs.String("api-url", defaultAPI, "Pingora Control API URL")

	if err := fs.Parse(args); err != nil {
		return domain.DeploymentConfig{}, "", err
	}
	if *contextPath == "" {
		return domain.DeploymentConfig{}, "", fmt.Errorf("flag --context / -c is required for deploy command")
	}

	cfg := domain.DeploymentConfig{
		ContextPath:    *contextPath,
		Name:           *name,
		Network:        *network,
		Port:           *port,
		HealthEndpoint: *healthEp,
		TimeoutSecs:    *timeout,
	}
	return cfg, *apiURL, nil
}

func ParseDeregisterFlags(args []string, defaultAPI string) (string, int, string, error) {
	fs := flag.NewFlagSet("deregister", flag.ContinueOnError)
	ip := fs.String("ip", "", "Upstream IP address (required)")
	port := fs.Int("port", 0, "Upstream port")
	apiURL := fs.String("api-url", defaultAPI, "Pingora Control API URL")

	if err := fs.Parse(args); err != nil {
		return "", 0, "", err
	}
	if *ip == "" {
		return "", 0, "", fmt.Errorf("flag --ip is required for deregister command")
	}
	return *ip, *port, *apiURL, nil
}

func ParseWatchFlags(args []string, defaultAPI string) (string, string, error) {
	fs := flag.NewFlagSet("watch", flag.ContinueOnError)
	network := fs.String("network", "edge-net", "Docker bridge network")
	apiURL := fs.String("api-url", defaultAPI, "Pingora Control API URL")

	if err := fs.Parse(args); err != nil {
		return "", "", err
	}
	return *network, *apiURL, nil
}
