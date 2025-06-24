package utils

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	vers "github.com/hdresearch/vers-sdk-go"
)

const (
	VersDir  = ".vers"
	HeadFile = "HEAD"
)

// GetCurrentHeadVM returns the VM ID from the current HEAD
func GetCurrentHeadVM() (string, error) {
	headFile := filepath.Join(VersDir, HeadFile)

	// Check if .vers directory and HEAD file exist
	if _, err := os.Stat(headFile); os.IsNotExist(err) {
		return "", fmt.Errorf("HEAD not found. Run 'vers init' first")
	}

	// Read HEAD file
	headData, err := os.ReadFile(headFile)
	if err != nil {
		return "", fmt.Errorf("error reading HEAD: %w", err)
	}

	// HEAD directly contains a VM ID or alias
	vmID := strings.TrimSpace(string(headData))

	if vmID == "" {
		return "", fmt.Errorf("HEAD is empty. Create a VM first with 'vers run'")
	}

	return vmID, nil
}

// SetHead sets the HEAD to point to a specific VM
func SetHead(vmID string) error {
	headFile := filepath.Join(VersDir, HeadFile)

	// Ensure .vers directory exists
	if err := os.MkdirAll(VersDir, 0755); err != nil {
		return fmt.Errorf("failed to create .vers directory: %w", err)
	}

	return os.WriteFile(headFile, []byte(vmID), 0644)
}

// ClearHead clears the HEAD file
func ClearHead() error {
	headFile := filepath.Join(VersDir, HeadFile)
	return os.WriteFile(headFile, []byte(""), 0644)
}

// CheckVMImpact checks if deleting a VM will affect HEAD
func CheckVMImpact(vmID string) string {
	headVM, err := GetCurrentHeadVM()
	if err != nil {
		return "" // No HEAD to worry about
	}

	if headVM == vmID {
		return "Current HEAD points to the VM being deleted. HEAD will be cleared."
	}

	return ""
}

// CheckClusterImpact checks if deleting a cluster will affect HEAD
func CheckClusterImpact(ctx context.Context, client *vers.Client, clusterID string) string {
	headVM, err := GetCurrentHeadVM()
	if err != nil {
		return "" // No HEAD to worry about
	}

	// Check if HEAD VM is in the cluster being deleted
	apiCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	vmResponse, err := client.API.Vm.Get(apiCtx, headVM)
	if err == nil && vmResponse.Data.ClusterID == clusterID {
		return "Current HEAD points to a VM in the cluster being deleted. HEAD will be cleared."
	}

	return ""
}

// CheckBatchImpact checks if any targets in a batch will affect HEAD
func CheckBatchImpact(ctx context.Context, client *vers.Client, vmIDs []string, clusterIDs []string) bool {
	headVM, err := GetCurrentHeadVM()
	if err != nil {
		return false
	}

	// Check direct VM matches
	for _, vmID := range vmIDs {
		if headVM == vmID {
			return true
		}
	}

	// Check cluster impacts
	if len(clusterIDs) > 0 {
		apiCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
		defer cancel()

		vmResponse, err := client.API.Vm.Get(apiCtx, headVM)
		if err != nil {
			return false
		}

		for _, clusterID := range clusterIDs {
			if vmResponse.Data.ClusterID == clusterID {
				return true
			}
		}
	}

	return false
}

// CleanupAfterDeletion clears HEAD if the VM it points to no longer exists
func CleanupAfterDeletion(ctx context.Context, client *vers.Client) {
	headVM, err := GetCurrentHeadVM()
	if err != nil {
		return
	}

	// Check if the VM still exists
	apiCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	_, err = client.API.Vm.Get(apiCtx, headVM)
	if err != nil {
		// VM no longer exists, clear HEAD
		ClearHead()
		fmt.Println("HEAD cleared (VM no longer exists)")
	}
}

// EnsureHeadExists checks if HEAD exists and is valid
func EnsureHeadExists(ctx context.Context, client *vers.Client) error {
	headVM, err := GetCurrentHeadVM()
	if err != nil {
		return err
	}

	// Verify the VM still exists
	apiCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	_, err = client.API.Vm.Get(apiCtx, headVM)
	if err != nil {
		ClearHead()
		return fmt.Errorf("HEAD pointed to non-existent VM, cleared HEAD")
	}

	return nil
}
