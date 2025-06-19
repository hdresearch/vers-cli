package utils

import (
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/hdresearch/vers-cli/internal/auth"
)

// GetNodeIPForVM makes a raw HTTP request to get the node IP from headers
func GetNodeIPForVM(vmID string) (string, error) {
	// Get API key for authentication
	apiKey, err := auth.GetAPIKey()
	if err != nil {
		return "", fmt.Errorf("failed to get API key: %w", err)
	}

	// Construct the URL using the same base URL logic as the SDK
	versUrl, err := auth.GetVersUrl()
	if err != nil {
		return "", fmt.Errorf("failed to get Vers URL: %w", err)
	}
	var baseURL string

	// Check if URL already has protocol
	if strings.HasPrefix(versUrl, "http://") || strings.HasPrefix(versUrl, "https://") {
		baseURL = versUrl
	} else {
		// Legacy: add protocol if missing
		if versUrl == "api.vers.sh" {
			baseURL = "https://" + versUrl
		} else {
			baseURL = "http://" + versUrl
		}
	}
	url := baseURL + "/api/vm/" + vmID

	// Create HTTP request
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return "", fmt.Errorf("failed to create request: %w", err)
	}

	// Add authentication header
	req.Header.Set("Authorization", "Bearer "+apiKey)

	// Make the request
	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("failed to make request: %w", err)
	}
	defer resp.Body.Close()

	// Get the node IP from headers
	nodeIP := resp.Header.Get("X-Node-IP")
	if nodeIP != "" && nodeIP != "unknown" {
		return nodeIP, nil
	}

	return "", fmt.Errorf("no node IP found in response headers")
}
