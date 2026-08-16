package api_utils

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os/exec"
	"strings"

	"dynamic_docker_apps/cli/domain"
)

type CommandRunner interface {
	RunCommand(name string, args ...string) ([]byte, error)
}

type RealCommandRunner struct{}

func (r RealCommandRunner) RunCommand(name string, args ...string) ([]byte, error) {
	return exec.Command(name, args...).CombinedOutput()
}

var runner CommandRunner = RealCommandRunner{}

func SetCommandRunner(r CommandRunner) {
	runner = r
}

func CheckApiServerHealth(apiURL string) error {
	endpoint := fmt.Sprintf("%s/health", strings.TrimRight(apiURL, "/"))
	if err := checkHealthViaDockerExec(endpoint); err != nil {
		return err
	}
	return nil
}

func checkHealthViaDockerExec(endpoint string) error {
	pyScript := fmt.Sprintf(`import urllib.request, sys
try:
    resp = urllib.request.urlopen('%s')
    sys.exit(0 if resp.status == 200 else 1)
except Exception as e:
    sys.stderr.write(str(e))
    sys.exit(1)`, endpoint)

	pyCmd := strings.ReplaceAll(pyScript, "\n", "; ")
	out, err := runner.RunCommand("docker", "exec", "pingora-lb", "python3", "-c", pyCmd)
	if err != nil {
		return parseDockerError(string(out), err)
	}
	return nil
}

func sendApiRequest(apiURL, method, path string, payload interface{}) ([]byte, error) {
	if err := CheckApiServerHealth(apiURL); err != nil {
		return nil, err
	}
	formattedPath := path
	if !strings.HasPrefix(formattedPath, "/") {
		formattedPath = "/" + formattedPath
	}
	url := fmt.Sprintf("%s%s", strings.TrimRight(apiURL, "/"), formattedPath)
	return sendApiRequestViaDockerExec(url, method, payload)
}

func sendApiRequestViaDockerExec(url, method string, payload interface{}) ([]byte, error) {
	jsonBytes, _ := json.Marshal(payload)
	payloadStr := strings.ReplaceAll(string(jsonBytes), "\"", "\\\"")

	pyScript := fmt.Sprintf(`import urllib.request, urllib.error, sys
try:
    data = """%s""".encode() if """%s""" != "null" else None
    req = urllib.request.Request('%s', data=data, headers={'Content-Type':'application/json'}, method='%s')
    with urllib.request.urlopen(req) as resp:
        print(resp.read().decode())
except urllib.error.HTTPError as e:
    sys.stderr.write(f"HTTP_ERROR:{e.code}:"+e.read().decode())
    sys.exit(1)
except Exception as e:
    sys.stderr.write(f"URL_ERROR:{str(e)}")
    sys.exit(1)`, payloadStr, payloadStr, url, method)

	pyCmd := strings.ReplaceAll(pyScript, "\n", "; ")
	out, err := runner.RunCommand("docker", "exec", "pingora-lb", "python3", "-c", pyCmd)
	if err != nil {
		return nil, parseExecOutputError(string(out))
	}
	return out, nil
}

func parseDockerError(out string, err error) error {
	if strings.Contains(out, "No such container") || strings.Contains(out, "is not running") {
		return fmt.Errorf("Pingora LB container 'pingora-lb' is not running in Docker")
	}
	return fmt.Errorf("Pingora Control API unreachable inside container: %s (%v)", strings.TrimSpace(out), err)
}

func parseExecOutputError(out string) error {
	outStr := strings.TrimSpace(out)
	if strings.HasPrefix(outStr, "HTTP_ERROR:") {
		parts := strings.SplitN(outStr, ":", 3)
		if len(parts) == 3 {
			code := 0
			_, _ = fmt.Sscanf(parts[1], "%d", &code)
			return parseApiError(code, []byte(parts[2]))
		}
	}
	if strings.Contains(outStr, "No such container") || strings.Contains(outStr, "is not running") {
		return fmt.Errorf("Pingora LB container 'pingora-lb' is not running in Docker")
	}
	return fmt.Errorf("API call failed: %s", outStr)
}

func parseApiError(statusCode int, body []byte) error {
	var apiErr domain.APIErrorResponse
	if err := json.Unmarshal(body, &apiErr); err == nil && apiErr.Error != "" {
		return fmt.Errorf("API error (%d): %s", apiErr.Code, apiErr.Error)
	}
	return fmt.Errorf("API error (%d): %s", statusCode, string(body))
}

func RegisterUpstream(apiURL string, target domain.UpstreamTarget) error {
	_, err := sendApiRequest(apiURL, http.MethodPost, "/upstreams", target)
	return err
}

func DeregisterUpstream(apiURL string, ip string, port int) error {
	payload := map[string]interface{}{"ip": ip}
	if port > 0 {
		payload["port"] = port
	}
	_, err := sendApiRequest(apiURL, http.MethodDelete, "/upstreams", payload)
	return err
}

func ListUpstreams(apiURL string) (string, error) {
	data, err := sendApiRequest(apiURL, http.MethodGet, "/upstreams", nil)
	if err != nil {
		return "", err
	}
	return string(data), nil
}
