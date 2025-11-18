package auth

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"

	"github.com/hdresearch/vers-cli/styles"
	"github.com/hdresearch/vers-sdk-go"
)

// SSHKeyResponse represents the API response for VM SSH key
type SSHKeyResponse struct {
	SSHPrivateKey string `json:"ssh_private_key"`
	SSHPort       int    `json:"ssh_port"`
}

// getSSHKeyPath returns the path to the SSH key file for a given VM
// Keys are stored in the system temp directory to avoid cluttering user directories
func getSSHKeyPath(vmID string) string {
	keysDir := filepath.Join(os.TempDir(), "vers-ssh-keys")
	return filepath.Join(keysDir, fmt.Sprintf("%s.key", vmID))
}

// fetchSSHKey makes a direct HTTP request to fetch the SSH key for a VM
func fetchSSHKey(ctx context.Context, client *vers.Client, vmID string) (*SSHKeyResponse, error) {
	// Get the API base URL and token
	versURL, err := GetVersUrl()
	if err != nil {
		return nil, fmt.Errorf("failed to get VERS URL: %w", err)
	}

	apiKey, err := GetAPIKey()
	if err != nil {
		return nil, fmt.Errorf("failed to get API key: %w", err)
	}

	// Build the URL
	url := fmt.Sprintf("%s://%s/api/v1/vm/%s/ssh_key", versURL.Scheme, versURL.Host, vmID)

	// Create the HTTP request
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	// Add headers
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", apiKey))
	req.Header.Set("Accept", "application/json")

	// Make the request
	httpClient := &http.Client{}
	resp, err := httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to make request: %w", err)
	}
	defer resp.Body.Close()

	// Check status code
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("API returned status %d: %s", resp.StatusCode, string(body))
	}

	// Parse the response
	var sshKeyResp SSHKeyResponse
	if err := json.NewDecoder(resp.Body).Decode(&sshKeyResp); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	return &sshKeyResp, nil
}

// GetOrCreateSSHKey retrieves the path to an SSH key, fetching and saving it if necessary.
// It returns the path to the key file and an error if any occurred.
func GetOrCreateSSHKey(vmID string, client *vers.Client, apiCtx context.Context) (string, error) {
	// Determine the path for storing the SSH key
	keyPath := getSSHKeyPath(vmID)

	// Check if SSH key already exists
	if _, err := os.Stat(keyPath); err == nil {
		// Key exists, return the path silently
		return keyPath, nil
	}

	// Fetch SSH key from API
	fmt.Printf("%s\n", styles.MutedTextStyle.Italic(true).Render("Fetching SSH key from API..."))
	resp, err := fetchSSHKey(apiCtx, client, vmID)
	if err != nil {
		return "", fmt.Errorf("failed to fetch SSH key: %w", err)
	}

	// Create the keys directory if it doesn't exist
	keysDir := filepath.Dir(keyPath)
	if err := os.MkdirAll(keysDir, 0755); err != nil {
		return "", fmt.Errorf("failed to create keys directory: %w", err)
	}

	// Save the SSH key to file with restrictive permissions (0600)
	if err := os.WriteFile(keyPath, []byte(resp.SSHPrivateKey), 0600); err != nil {
		return "", fmt.Errorf("failed to save SSH key: %w", err)
	}

	successStyle := styles.PrimaryTextStyle.Foreground(styles.TerminalGreen).Bold(true)
	fmt.Printf("%s\n", successStyle.Render("✓ SSH key cached"))
	return keyPath, nil
}
