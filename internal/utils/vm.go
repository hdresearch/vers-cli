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
}

// ResolveVMIdentifier takes a VM ID and returns the VM info
// Note: Alias lookups are no longer supported in the new SDK
func ResolveVMIdentifier(ctx context.Context, client *vers.Client, vmID string) (*VMInfo, error) {
	vms, err := client.Vm.List(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to list VMs: %w", err)
	}

	for _, vm := range *vms {
		if vm.VmID == vmID {
			return &VMInfo{
				ID:          vm.VmID,
				DisplayName: vm.VmID,
			}, nil
		}
	}

	return nil, fmt.Errorf("VM '%s' not found", vmID)
}

// CreateVMInfoFromVM creates VMInfo from a Vm struct
func CreateVMInfoFromVM(vm vers.Vm) *VMInfo {
	return &VMInfo{
		ID:          vm.VmID,
		DisplayName: vm.VmID,
	}
}

