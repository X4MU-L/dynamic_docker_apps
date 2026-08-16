package domain

import (
	"fmt"
	"math/rand"
	"time"
)

const DefaultDomainSuffix = "edge.local"

type DeploymentConfig struct {
	ContextPath    string
	Image          string
	Username       string
	Password       string
	Name           string
	Network        string
	DomainSuffix   string
	Port           int
	HealthEndpoint string
	TimeoutSecs    int
	Replicas       int
}

type UpstreamTarget struct {
	IP             string `json:"ip"`
	Port           int    `json:"port"`
	SNIName        string `json:"sni_name"`
	HealthEndpoint string `json:"health_endpoint"`
}

type BackendItem struct {
	IP             string `json:"ip"`
	Port           int    `json:"port"`
	SNIName        string `json:"sni_name"`
	HealthEndpoint string `json:"health_endpoint"`
}

type APIErrorResponse struct {
	Error string `json:"error"`
	Code  int    `json:"code"`
}

func NewUpstreamTarget(ip string, port int, containerName string, domainSuffix string, health string) UpstreamTarget {
	if domainSuffix == "" {
		domainSuffix = DefaultDomainSuffix
	}
	sni := fmt.Sprintf("%s.%s", containerName, domainSuffix)
	if health == "" {
		health = "/health"
	}
	return UpstreamTarget{
		IP:             ip,
		Port:           port,
		SNIName:        sni,
		HealthEndpoint: health,
	}
}

func GenerateContainerName(prefix string) string {
	if prefix == "" {
		prefix = "app"
	}
	r := rand.New(rand.NewSource(time.Now().UnixNano()))
	const charset = "abcdefghijklmnopqrstuvwxyz0123456789"
	b := make([]byte, 6)
	for i := range b {
		b[i] = charset[r.Intn(len(charset))]
	}
	return fmt.Sprintf("%s-%s", prefix, string(b))
}
