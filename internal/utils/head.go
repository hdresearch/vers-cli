package utils

import (
    "context"
    "errors"
    "fmt"
    "os"
    "path/filepath"
    "strings"

    vers "github.com/hdresearch/vers-sdk-go"
)

const (
	VersDir  = ".vers"
	HeadFile = "HEAD"
)

var (
    ErrHeadNotFound = errors.New("head not found")
    ErrHeadEmpty    = errors.New("head empty")
)

// GetCurrentHeadVM returns the VM ID from the current HEAD
func GetCurrentHeadVM() (string, error) {
    headFile := filepath.Join(VersDir, HeadFile)

    // Check if .vers directory and HEAD file exist
    if _, err := os.Stat(headFile); os.IsNotExist(err) {
        return "", ErrHeadNotFound
    }

	// Read HEAD file
	headData, err := os.ReadFile(headFile)
	if err != nil {
		return "", fmt.Errorf("error reading HEAD: %w", err)
	}

	// HEAD directly contains a VM ID or alias
	vmID := strings.TrimSpace(string(headData))

    if vmID == "" {
        return "", ErrHeadEmpty
    }

	return vmID, nil
}

// SetHead sets the HEAD to point to a specific VM ID (always stores ID, never alias)
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

// GetCurrentHeadVMInfo returns both the HEAD VM ID and its display information
func GetCurrentHeadVMInfo(ctx context.Context, client *vers.Client) (*VMInfo, error) {
	headVM, err := GetCurrentHeadVM()
	if err != nil {
		return nil, err
	}

	return ResolveVMIdentifier(ctx, client, headVM)
}

// SetHeadFromIdentifier resolves a VM identifier to an ID and sets HEAD
func SetHeadFromIdentifier(ctx context.Context, client *vers.Client, identifier string) (*VMInfo, error) {
	vmInfo, err := ResolveVMIdentifier(ctx, client, identifier)
	if err != nil {
		return nil, err
	}

	if err := SetHead(vmInfo.ID); err != nil {
		return nil, fmt.Errorf("failed to update HEAD: %w", err)
	}

	return vmInfo, nil
}

// CheckVMImpactsHead checks if a specific VM deletion will affect HEAD
func CheckVMImpactsHead(vmID string) bool {
	headVM, err := GetCurrentHeadVM()
	if err != nil {
		return false
	}
	return headVM == vmID
}

// ConfirmVMHeadImpact checks and confirms HEAD impact for a single VM deletion
// Confirmation prompts were moved to handlers using the shared Prompter.

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
