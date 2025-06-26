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

// VMInfo contains both ID and display name for a VM
type VMInfo struct {
	ID          string
	DisplayName string
	State       string
}

// ClusterInfo contains both ID and display name for a cluster
type ClusterInfo struct {
	ID          string
	DisplayName string
	VmCount     int
}

// ResolveVMIdentifier takes a VM ID or alias and returns the VM ID and display info
// This ensures all API calls use IDs while providing good UX with display names
func ResolveVMIdentifier(ctx context.Context, client *vers.Client, identifier string) (*VMInfo, error) {
	response, err := client.API.Vm.Get(ctx, identifier)
	if err != nil {
		return nil, fmt.Errorf("VM '%s' not found: %w", identifier, err)
	}

	vm := response.Data
	displayName := vm.Alias
	if displayName == "" {
		displayName = vm.ID
	}

	return &VMInfo{
		ID:          vm.ID,
		DisplayName: displayName,
		State:       string(vm.State),
	}, nil
}

// ResolveClusterIdentifier takes a cluster ID or alias and returns the cluster ID and display info
func ResolveClusterIdentifier(ctx context.Context, client *vers.Client, identifier string) (*ClusterInfo, error) {
	response, err := client.API.Cluster.Get(ctx, identifier)
	if err != nil {
		return nil, fmt.Errorf("cluster '%s' not found: %w", identifier, err)
	}

	cluster := response.Data
	displayName := cluster.Alias
	if displayName == "" {
		displayName = cluster.ID
	}

	return &ClusterInfo{
		ID:          cluster.ID,
		DisplayName: displayName,
		VmCount:     int(cluster.VmCount),
	}, nil
}

// CreateVMInfoFromGetResponse creates VMInfo from a Get API response
// Use this when you already have VM data from Get endpoint to avoid extra API calls
func CreateVMInfoFromGetResponse(vm vers.APIVmGetResponseData) *VMInfo {
	displayName := vm.Alias
	if displayName == "" {
		displayName = vm.ID
	}

	return &VMInfo{
		ID:          vm.ID,
		DisplayName: displayName,
		State:       string(vm.State),
	}
}

// CreateVMInfoFromUpdateResponse creates VMInfo from an Update API response
// Use this when you already have VM data from Update endpoint to avoid extra API calls
func CreateVMInfoFromUpdateResponse(vm vers.APIVmUpdateResponseData) *VMInfo {
	displayName := vm.Alias
	if displayName == "" {
		displayName = vm.ID
	}

	return &VMInfo{
		ID:          vm.ID,
		DisplayName: displayName,
		State:       string(vm.State),
	}
}

// CreateClusterInfoFromListResponse creates ClusterInfo from a List API response item
// Use this when you already have cluster data from List endpoint to avoid extra API calls
func CreateClusterInfoFromListResponse(cluster vers.APIClusterListResponseData) *ClusterInfo {
	displayName := cluster.Alias
	if displayName == "" {
		displayName = cluster.ID
	}

	return &ClusterInfo{
		ID:          cluster.ID,
		DisplayName: displayName,
		VmCount:     int(cluster.VmCount),
	}
}

// GetCurrentHeadVMInfo returns both the HEAD VM ID and its display information
// WARNING: This makes an API call! Use GetCurrentHeadVM() + existing API response when possible
func GetCurrentHeadVMInfo(ctx context.Context, client *vers.Client) (*VMInfo, error) {
	headVM, err := GetCurrentHeadVM()
	if err != nil {
		return nil, err
	}

	return ResolveVMIdentifier(ctx, client, headVM)
}

// SetHeadFromIdentifier resolves a VM identifier to an ID and sets HEAD
// This ensures HEAD always contains IDs regardless of user input
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

// CheckVMImpactsHead checks if a specific VM deletion will affect HEAD
func CheckVMImpactsHead(vmID string) bool {
	headVM, err := GetCurrentHeadVM()
	if err != nil {
		return false
	}
	return headVM == vmID
}

// CheckClusterImpactsHead checks if a specific cluster deletion will affect HEAD
func CheckClusterImpactsHead(ctx context.Context, client *vers.Client, clusterID string) bool {
	headVM, err := GetCurrentHeadVM()
	if err != nil {
		return false
	}

	apiCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	vmResponse, err := client.API.Vm.Get(apiCtx, headVM)
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
