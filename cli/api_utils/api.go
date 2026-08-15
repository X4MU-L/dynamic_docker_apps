package api_utils

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"dynamic_docker_apps/cli/domain"
)

var client = &http.Client{Timeout: 5 * time.Second}

func CheckApiServerHealth(apiURL string) error {
	endpoint := fmt.Sprintf("%s/health", strings.TrimRight(apiURL, "/"))
	resp, err := client.Get(endpoint)
	if err != nil {
		return fmt.Errorf("Pingora Control API server is unreachable at %s (%v)", apiURL, err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("Pingora Control API health check returned status %s", resp.Status)
	}
	return nil
}

func sendApiRequest(apiURL, method, path string, payload interface{}) ([]byte, error) {
	if err := CheckApiServerHealth(apiURL); err != nil {
		return nil, err
	}
	url := fmt.Sprintf("%s%s", strings.TrimRight(apiURL, "/"), path)
	req, err := createHttpRequest(method, url, payload)
	if err != nil {
		return nil, err
	}

	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("HTTP %s failed for %s: %w", method, url, err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, parseApiError(resp.StatusCode, respBody)
	}
	return respBody, nil
}

func createHttpRequest(method, url string, payload interface{}) (*http.Request, error) {
	var bodyReader io.Reader
	if payload != nil {
		data, err := json.Marshal(payload)
		if err != nil {
			return nil, err
		}
		bodyReader = bytes.NewBuffer(data)
	}
	req, err := http.NewRequest(method, url, bodyReader)
	if err != nil {
		return nil, err
	}
	if payload != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	return req, nil
}

func parseApiError(statusCode int, body []byte) error {
	var apiErr domain.APIErrorResponse
	if err := json.Unmarshal(body, &apiErr); err == nil && apiErr.Error != "" {
		return fmt.Errorf("API error (%d): %s", apiErr.Code, apiErr.Error)
	}
	return fmt.Errorf("API error (%d): %s", statusCode, string(body))
}

func RegisterUpstream(apiURL string, payload domain.UpstreamTarget) error {
	_, err := sendApiRequest(apiURL, http.MethodPost, "/upstreams", payload)
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
