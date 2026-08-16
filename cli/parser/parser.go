package parser

import (
	"flag"
	"fmt"
	"strings"

	"dynamic_docker_apps/cli/domain"
)

func isHelpRequested(args []string) bool {
	for _, arg := range args {
		if arg == "-h" || arg == "--help" || arg == "help" {
			return true
		}
	}
	return false
}

func ParseDeployFlags(args []string, defaultAPI string) (domain.DeploymentConfig, string, error) {
	if isHelpRequested(args) {
		PrintDeployHelp()
		return domain.DeploymentConfig{}, "", flag.ErrHelp
	}
	fs := flag.NewFlagSet("deploy", flag.ContinueOnError)
	contextPath := fs.String("context", "", "Path to Docker context build directory")
	fs.StringVar(contextPath, "c", "", "Path to Docker context build directory (shorthand)")
	image := fs.String("image", "", "Pre-built Docker image tag")
	fs.StringVar(image, "i", "", "Pre-built Docker image tag (shorthand)")
	username := fs.String("username", "", "Registry username")
	fs.StringVar(username, "u", "", "Registry username (shorthand)")
	password := fs.String("password", "", "Registry password")
	name := fs.String("name", "", "Container instance name")
	fs.StringVar(name, "n", "", "Container instance name (shorthand)")
	replicas := fs.Int("replicas", 1, "Number of container replicas")
	fs.IntVar(replicas, "r", 1, "Number of container replicas (shorthand)")
	domainSuffix := fs.String("domain", domain.DefaultDomainSuffix, "Domain suffix")
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
	nameLower := strings.ToLower(strings.TrimSpace(*name))
	domainLower := strings.ToLower(strings.TrimSpace(*domainSuffix))

	if err := validateDeployArgs(*contextPath, *image, nameLower, domainLower, *replicas); err != nil {
		return domain.DeploymentConfig{}, "", err
	}

	cfg := domain.DeploymentConfig{
		ContextPath:    *contextPath,
		Image:          *image,
		Username:       *username,
		Password:       *password,
		Name:           nameLower,
		Replicas:       *replicas,
		Network:        *network,
		DomainSuffix:   domainLower,
		Port:           *port,
		HealthEndpoint: *healthEp,
		TimeoutSecs:    *timeout,
	}
	return cfg, *apiURL, nil
}

func validateDeployArgs(contextPath, image, name, domainSuffix string, replicas int) error {
	if contextPath == "" && image == "" {
		return fmt.Errorf("either --image (-i) or --context (-c) must be specified for deploy command")
	}
	if replicas < 1 {
		return fmt.Errorf("replicas count must be at least 1, got %d", replicas)
	}
	if name != "" && !isValidUrlSafeName(name) {
		return fmt.Errorf("container name '%s' is not URL-safe", name)
	}
	if domainSuffix != "" && !isValidDomainSuffix(domainSuffix) {
		return fmt.Errorf("domain suffix '%s' is not URL-safe", domainSuffix)
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

func PrintDeployHelp() {
	fmt.Println("Usage: deployer deploy [flags]")
	fmt.Println("\nDeploy containerized app replicas to edge-net and register with Pingora LB.")
	fmt.Println("\nFlags:")
	fmt.Println("  -c, --context          Path to Docker context build directory")
	fmt.Println("  -i, --image            Pre-built Docker image tag (e.g. nginx:latest)")
	fmt.Println("  -n, --name             Container instance name prefix")
	fmt.Println("  -r, --replicas         Number of container replicas to deploy (default 1)")
	fmt.Println("  -d, --domain           Domain suffix for hostname/SNI (default edge.local)")
	fmt.Println("  -u, --username         Registry authentication username")
	fmt.Println("      --password         Registry authentication password")
	fmt.Println("  -p, --port             Container app port (default 8080)")
	fmt.Println("      --network          Docker bridge network (default edge-net)")
	fmt.Println("      --health-endpoint  Health probe path (default /health)")
	fmt.Println("      --timeout          Health probe timeout seconds (default 30)")
	fmt.Println("      --api-url          Pingora Control API URL (default http://localhost:8081)")
}

func ParseDeregisterFlags(args []string, defaultAPI string) (string, string, int, bool, int, string, string, error) {
	if isHelpRequested(args) {
		PrintDeregisterHelp()
		return "", "", 0, false, 0, "", "", flag.ErrHelp
	}
	fs := flag.NewFlagSet("deregister", flag.ContinueOnError)
	name := fs.String("name", "", "Container instance name")
	fs.StringVar(name, "n", "", "Container instance name (shorthand)")
	ip := fs.String("ip", "", "Upstream IP address")
	port := fs.Int("port", 0, "Upstream port")
	fs.IntVar(port, "p", 0, "Upstream port (shorthand)")
	stop := fs.Bool("stop", false, "Stop and remove container after deregistering")
	fs.BoolVar(stop, "s", false, "Stop and remove container after deregistering (shorthand)")
	drainTimeout := fs.Int("drain-timeout", 15, "Drain timeout in seconds (0 for immediate force eviction)")
	fs.IntVar(drainTimeout, "t", 15, "Drain timeout in seconds (shorthand)")
	network := fs.String("network", "edge-net", "Docker bridge network")
	apiURL := fs.String("api-url", defaultAPI, "Pingora Control API URL")

	if err := fs.Parse(args); err != nil {
		return "", "", 0, false, 0, "", "", err
	}
	nameVal := strings.ToLower(strings.TrimSpace(*name))
	ipVal := strings.TrimSpace(*ip)

	if nameVal == "" && ipVal == "" {
		return "", "", 0, false, 0, "", "", fmt.Errorf("either --name (-n) or --ip must be specified for deregister command")
	}
	return nameVal, ipVal, *port, *stop, *drainTimeout, *network, *apiURL, nil
}

func PrintDeregisterHelp() {
	fmt.Println("Usage: deployer deregister [flags]")
	fmt.Println("\nEvict an upstream from Pingora LB by container name or IP address.")
	fmt.Println("\nFlags:")
	fmt.Println("  -n, --name           Container name to resolve IP and evict")
	fmt.Println("      --ip             Upstream IP address to evict directly")
	fmt.Println("  -p, --port           Upstream port")
	fmt.Println("  -s, --stop           Stop and remove container after deregistering (default false)")
	fmt.Println("  -t, --drain-timeout  Drain timeout in seconds, 0 for force eviction (default 15)")
	fmt.Println("      --network        Docker bridge network (default edge-net)")
	fmt.Println("      --api-url        Pingora Control API URL (default http://localhost:8081)")
}

func ParseWatchFlags(args []string, defaultAPI string) (string, string, error) {
	if isHelpRequested(args) {
		PrintWatchHelp()
		return "", "", flag.ErrHelp
	}
	fs := flag.NewFlagSet("watch", flag.ContinueOnError)
	network := fs.String("network", "edge-net", "Docker bridge network")
	apiURL := fs.String("api-url", defaultAPI, "Pingora Control API URL")

	if err := fs.Parse(args); err != nil {
		return "", "", err
	}
	return *network, *apiURL, nil
}

func PrintWatchHelp() {
	fmt.Println("Usage: deployer watch [flags]")
	fmt.Println("\nListen for Docker container death events and auto-evict endpoints from Pingora LB.")
	fmt.Println("\nFlags:")
	fmt.Println("  --network  Docker bridge network (default edge-net)")
	fmt.Println("  --api-url  Pingora Control API URL")
}

func PrintListHelp() {
	fmt.Println("Usage: deployer list [flags]")
	fmt.Println("\nList all active upstreams registered with Pingora LB.")
	fmt.Println("\nFlags:")
	fmt.Println("  --api-url  Pingora Control API URL")
}
