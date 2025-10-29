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
		// Key exists, return the path
		fmt.Println(s.HeadStatus.Render(fmt.Sprintf("Using existing SSH key from %s", keyPath)))
		return keyPath, nil
	}

	// TODO: SSH key retrieval method needs to be reimplemented with new SDK
	// The GetSSHKey method has been removed from the SDK
	// For now, return an error indicating this functionality is not available
	return "", fmt.Errorf("SSH key retrieval is not yet supported in the new SDK version")
}
