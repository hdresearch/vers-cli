package auth

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"github.com/hdresearch/vers-cli/styles"
	"github.com/hdresearch/vers-sdk-go"
)

// getSSHKeyPath returns the path to the SSH key file for a given VM
func getSSHKeyPath(vmID string) string {
	versDir := ".vers"
	keysDir := filepath.Join(versDir, "keys")
	return filepath.Join(keysDir, fmt.Sprintf("%s.key", vmID))
}

// GetOrCreateSSHKey retrieves the path to an SSH key, fetching and saving it if necessary.
// It returns the path to the key file and an error if any occurred.
func GetOrCreateSSHKey(vmID string, client *vers.Client, apiCtx context.Context) (string, error) {
	s := styles.NewStatusStyles()
	// Determine the path for storing the SSH key
	keyPath := getSSHKeyPath(vmID)

	// Check if SSH key already exists
	if _, err := os.Stat(keyPath); err == nil {
		// Key exists, return the path silently (no output for cached keys)
		return keyPath, nil
	}

	// Key doesn't exist, need to fetch it
	fmt.Println(s.HeadStatus.Render("Fetching SSH key..."))

	keysDir := filepath.Dir(keyPath)
	if err := os.MkdirAll(keysDir, 0755); err != nil {
		return "", fmt.Errorf(s.NoData.Render("failed to create keys directory: %w"), err)
	}

	response, err := client.API.Vm.GetSSHKey(apiCtx, vmID)
	if err != nil {
		return "", fmt.Errorf(s.NoData.Render("failed to get SSH key: %w"), err)
	}
	sshKeyBytes := response.Data

	if err := os.WriteFile(keyPath, []byte(sshKeyBytes), 0600); err != nil {
		return "", fmt.Errorf(s.NoData.Render("failed to write key file: %w"), err)
	}

	fmt.Println(s.HeadStatus.Render("SSH key saved and ready"))
	return keyPath, nil
}
