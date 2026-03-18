package auth

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"github.com/hdresearch/vers-sdk-go"
)

// SSHKeyResponse represents the API response for VM SSH key
type SSHKeyResponse struct {
	SSHPrivateKey string `json:"ssh_private_key"`
	SSHPort       int    `json:"ssh_port"`
}

// getSSHKeyPath returns the path to the SSH key file for a given VM
func getSSHKeyPath(vmID string) string {
	keysDir := filepath.Join(os.TempDir(), "vers-ssh-keys")
	return filepath.Join(keysDir, fmt.Sprintf("%s.key", vmID))
}

func fetchSSHKey(ctx context.Context, client *vers.Client, vmID string) (*vers.VmSSHKeyResponse, error) {
	resp, err := client.Vm.GetSSHKey(ctx, vmID)
	if err != nil {
		return nil, err
	}
	return resp, nil
}

// GetOrCreateSSHKey retrieves the path to an SSH key, fetching and saving it if necessary.
func GetOrCreateSSHKey(vmID string, client *vers.Client, apiCtx context.Context) (string, error) {
	keyPath := getSSHKeyPath(vmID)

	if _, err := os.Stat(keyPath); err == nil {
		return keyPath, nil
	}

	fmt.Println("Fetching SSH key from API...")
	resp, err := fetchSSHKey(apiCtx, client, vmID)
	if err != nil {
		return "", fmt.Errorf("failed to fetch SSH key: %w", err)
	}

	keysDir := filepath.Dir(keyPath)
	if err := os.MkdirAll(keysDir, 0755); err != nil {
		return "", fmt.Errorf("failed to create keys directory: %w", err)
	}

	if err := os.WriteFile(keyPath, []byte(resp.SSHPrivateKey), 0600); err != nil {
		return "", fmt.Errorf("failed to save SSH key: %w", err)
	}

	fmt.Println("✓ SSH key cached")
	return keyPath, nil
}
