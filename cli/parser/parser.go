package parser

import (
	"flag"
	"fmt"
	"strings"

	"dynamic_docker_apps/cli/domain"
)

func ParseDeployFlags(args []string, defaultAPI string) (domain.DeploymentConfig, string, error) {
	fs := flag.NewFlagSet("deploy", flag.ContinueOnError)
	contextPath := fs.String("context", "", "Path to Docker context build directory")
	fs.StringVar(contextPath, "c", "", "Path to Docker context build directory (shorthand)")
	name := fs.String("name", "", "Container instance name")
	fs.StringVar(name, "n", "", "Container instance name (shorthand)")
	domainSuffix := fs.String("domain", domain.DefaultDomainSuffix, "Domain suffix for hostname/SNI")
	fs.StringVar(domainSuffix, "d", domain.DefaultDomainSuffix, "Domain suffix (shorthand)")
	port := fs.Int("port", 8080, "Container app port")
	fs.IntVar(port, "p", 8080, "Container app port (shorthand)")
	network := fs.String("network", "edge-net", "Docker bridge network")
	healthEp := fs.String("health-endpoint", "/health", "Health probe path")
	timeout := fs.Int("timeout", 30, "Health probe timeout seconds")
	apiURL := fs.String("api-url", defaultAPI, "Pingora Control API URL")

	if err := fs.Parse(args); err != nil {
		return domain.DeploymentConfig{}, "", err
	}
	if err := validateDeployArgs(*contextPath, *name, *domainSuffix); err != nil {
		return domain.DeploymentConfig{}, "", err
	}

	cfg := domain.DeploymentConfig{
		ContextPath:    *contextPath,
		Name:           *name,
		Network:        *network,
		DomainSuffix:   *domainSuffix,
		Port:           *port,
		HealthEndpoint: *healthEp,
		TimeoutSecs:    *timeout,
	}
	return cfg, *apiURL, nil
}

func validateDeployArgs(contextPath, name, domainSuffix string) error {
	if contextPath == "" {
		return fmt.Errorf("flag --context / -c is required for deploy command")
	}
	if name != "" && !isValidUrlSafeName(name) {
		return fmt.Errorf("container name '%s' is not URL-safe (must contain letters, numbers, and hyphens)", name)
	}
	if domainSuffix != "" && !isValidDomainSuffix(domainSuffix) {
		return fmt.Errorf("domain suffix '%s' is not URL-safe (e.g. edge.local)", domainSuffix)
	}
	return nil
}

func isValidUrlSafeName(s string) bool {
	if len(s) == 0 || len(s) > 63 || strings.HasPrefix(s, "-") || strings.HasSuffix(s, "-") {
		return false
	}
	for _, ch := range s {
		if !((ch >= 'a' && ch <= 'z') || (ch >= 'A' && ch <= 'Z') || (ch >= '0' && ch <= '9') || ch == '-') {
			return false
		}
	}
	return true
}

func isValidDomainSuffix(s string) bool {
	labels := strings.Split(s, ".")
	for _, label := range labels {
		if !isValidUrlSafeName(label) {
			return false
		}
	}
	return true
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
