package utils

import (
	"context"
	"encoding/json"
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

// HeadInfo represents the information stored in the HEAD file
type HeadInfo struct {
	ID    string `json:"id"`
	Alias string `json:"alias,omitempty"`
}

// DisplayName returns the alias if available, otherwise the ID
func (h *HeadInfo) DisplayName() string {
	if h.Alias != "" {
		return h.Alias
	}
	return h.ID
}

// GetCurrentHeadVM returns the VM ID from the current HEAD (for backward compatibility)
func GetCurrentHeadVM() (string, error) {
	headInfo, err := GetCurrentHead()
	if err != nil {
		return "", err
	}
	return headInfo.ID, nil
}

// GetCurrentHead returns the complete HEAD information (ID and alias)
func GetCurrentHead() (*HeadInfo, error) {
	headFile := filepath.Join(VersDir, HeadFile)

	// Check if .vers directory and HEAD file exist
	if _, err := os.Stat(headFile); os.IsNotExist(err) {
		return nil, fmt.Errorf("HEAD not found. Run 'vers init' first")
	}

	// Read HEAD file
	headData, err := os.ReadFile(headFile)
	if err != nil {
		return nil, fmt.Errorf("error reading HEAD: %w", err)
	}

	content := strings.TrimSpace(string(headData))
	if content == "" {
		return nil, fmt.Errorf("HEAD is empty. Create a VM first with 'vers run'")
	}

	// Try to parse as JSON first (new format)
	var headInfo HeadInfo
	if err := json.Unmarshal([]byte(content), &headInfo); err == nil {
		if headInfo.ID == "" {
			return nil, fmt.Errorf("HEAD contains invalid data")
		}
		return &headInfo, nil
	}

	// Fallback: treat as plain VM ID (old format for backward compatibility)
	return &HeadInfo{ID: content, Alias: ""}, nil
}

// SetHead sets the HEAD to point to a specific VM ID (legacy - no alias)
func SetHead(vmID string) error {
	return SetHeadWithAlias(vmID, "")
}

// SetHeadWithAlias sets the HEAD to point to a specific VM with both ID and alias
func SetHeadWithAlias(vmID, alias string) error {
	headFile := filepath.Join(VersDir, HeadFile)

	// Ensure .vers directory exists
	if err := os.MkdirAll(VersDir, 0755); err != nil {
		return fmt.Errorf("failed to create .vers directory: %w", err)
	}

	headInfo := HeadInfo{
		ID:    vmID,
		Alias: alias,
	}

	data, err := json.Marshal(headInfo)
	if err != nil {
		return fmt.Errorf("failed to marshal HEAD info: %w", err)
	}

	return os.WriteFile(headFile, data, 0644)
}

// ClearHead clears the HEAD file
func ClearHead() error {
	headFile := filepath.Join(VersDir, HeadFile)
	return os.WriteFile(headFile, []byte(""), 0644)
}

// GetCurrentHeadVMInfo returns both the HEAD VM ID and its display information
func GetCurrentHeadVMInfo(ctx context.Context, client *vers.Client) (*VMInfo, error) {
	headInfo, err := GetCurrentHead()
	if err != nil {
		return nil, err
	}

	return ResolveVMIdentifier(ctx, client, headInfo.ID)
}

// GetCurrentHeadDisplayName returns the display name (alias or ID) for HEAD
func GetCurrentHeadDisplayName() (string, error) {
	headInfo, err := GetCurrentHead()
	if err != nil {
		return "", err
	}
	return headInfo.DisplayName(), nil
}

// SetHeadFromIdentifier resolves a VM identifier to an ID and sets HEAD with alias
func SetHeadFromIdentifier(ctx context.Context, client *vers.Client, identifier string) (*VMInfo, error) {
	vmInfo, err := ResolveVMIdentifier(ctx, client, identifier)
	if err != nil {
		return nil, err
	}

	if err := SetHeadWithAlias(vmInfo.ID, vmInfo.Alias); err != nil {
		return nil, fmt.Errorf("failed to update HEAD: %w", err)
	}

	return vmInfo, nil
}

// CheckVMImpactsHead checks if a specific VM deletion will affect HEAD
func CheckVMImpactsHead(vmID string) bool {
	headInfo, err := GetCurrentHead()
	if err != nil {
		return false
	}
	return headInfo.ID == vmID
}

// CheckClusterImpactsHead checks if a specific cluster deletion will affect HEAD
func CheckClusterImpactsHead(ctx context.Context, client *vers.Client, clusterID string) bool {
	headInfo, err := GetCurrentHead()
	if err != nil {
		return false
	}

	apiCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	vmResponse, err := client.API.Vm.Get(apiCtx, headInfo.ID)
	if err != nil {
		return false
	}

	return vmResponse.Data.ClusterID == clusterID
}

// ConfirmVMHeadImpact checks and confirms HEAD impact for a single VM deletion
func ConfirmVMHeadImpact(vmID string, s *styles.KillStyles) bool {
	if !CheckVMImpactsHead(vmID) {
		return true // No impact, proceed
	}

	fmt.Println(s.Warning.Render("Warning: This will affect the current HEAD"))
	return AskConfirmation()
}

// ConfirmClusterHeadImpact checks and confirms HEAD impact for a single cluster deletion
func ConfirmClusterHeadImpact(ctx context.Context, client *vers.Client, clusterID string, s *styles.KillStyles) bool {
	if !CheckClusterImpactsHead(ctx, client, clusterID) {
		return true // No impact, proceed
	}

	fmt.Println(s.Warning.Render("Warning: This will affect the current HEAD"))
	return AskConfirmation()
}

// CleanupAfterDeletion clears HEAD if any of the deleted VM IDs match current HEAD
func CleanupAfterDeletion(deletedVMIDs []string) bool {
	headInfo, err := GetCurrentHead()
	if err != nil {
		return false
	}

	for _, deletedID := range deletedVMIDs {
		if headInfo.ID == deletedID {
			ClearHead()
			return true
		}
	}

	return false
}
