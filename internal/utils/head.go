package utils

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/hdresearch/vers-cli/styles"
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

// ConfirmHeadImpact checks and confirms HEAD impact for VM or cluster deletions
// This consolidates the duplicated logic from both processors
func ConfirmHeadImpact(ctx context.Context, client *vers.Client, vmIDs []string, clusterIDs []string, s *styles.KillStyles) bool {
	if !CheckBatchImpact(ctx, client, vmIDs, clusterIDs) {
		return true // No impact, proceed
	}

	// Determine appropriate warning message
	var message string
	totalItems := len(vmIDs) + len(clusterIDs)

	if totalItems == 1 {
		message = "Warning: This will affect the current HEAD"
	} else {
		if len(vmIDs) > 0 && len(clusterIDs) > 0 {
			message = "Warning: Some targets will affect the current HEAD"
		} else if len(vmIDs) > 1 {
			message = "Warning: Some VMs will affect the current HEAD"
		} else {
			message = "Warning: Some clusters will affect the current HEAD"
		}
	}

	fmt.Println(s.Warning.Render(message))
	return AskConfirmation()
}

// CleanupAfterDeletion clears HEAD if any of the deleted VM IDs match current HEAD
func CleanupAfterDeletion(deletedVMIDs []string) bool {
	headVM, err := GetCurrentHeadVM()
	if err != nil {
		return false
	}

	for _, deletedID := range deletedVMIDs {
		if headVM == deletedID {
			ClearHead()
			return true
		}
	}

	return false
}
