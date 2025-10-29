package status

import (
	"context"
	"fmt"

	vers "github.com/hdresearch/vers-sdk-go"
)

// ListVMs returns all VMs for the current user
func ListVMs(ctx context.Context, client *vers.Client) ([]vers.Vm, error) {
	resp, err := client.Vm.List(ctx)
	if err != nil {
		return nil, err
	}
	return *resp, nil
}

// GetVM retrieves a single VM by ID
func GetVM(ctx context.Context, client *vers.Client, vmID string) (*vers.Vm, error) {
	vms, err := ListVMs(ctx, client)
	if err != nil {
		return nil, err
	}
	for _, vm := range vms {
		if vm.VmID == vmID {
			return &vm, nil
		}
	}
	return nil, fmt.Errorf("VM with ID %s not found", vmID)
}
