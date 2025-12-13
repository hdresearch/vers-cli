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

// ResolveVMIdentifier takes a VM ID or alias and returns the VM info
// Aliases are resolved locally from ~/.vers/aliases.json
func ResolveVMIdentifier(ctx context.Context, client *vers.Client, identifier string) (*VMInfo, error) {
	// Resolve alias to VM ID if applicable
	vmID := ResolveAlias(identifier)

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

	return nil, fmt.Errorf("VM '%s' not found", identifier)
}

// CreateVMInfoFromVM creates VMInfo from a Vm struct
func CreateVMInfoFromVM(vm vers.Vm) *VMInfo {
	return &VMInfo{
		ID:          vm.VmID,
		DisplayName: vm.VmID,
	}
}
