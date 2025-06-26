package utils

import (
	"context"
	"fmt"

	vers "github.com/hdresearch/vers-sdk-go"
)

// VMInfo contains both ID and display name for a VM
type VMInfo struct {
	ID          string
	DisplayName string
	State       string
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
