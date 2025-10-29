package tree

import (
	"context"
	"fmt"

	vers "github.com/hdresearch/vers-sdk-go"
)

// GetVMByID fetches a VM by ID
func GetVMByID(ctx context.Context, client *vers.Client, vmID string) (*vers.Vm, error) {
	vms, err := client.Vm.List(ctx)
	if err != nil {
		return nil, err
	}
	for _, vm := range *vms {
		if vm.VmID == vmID {
			return &vm, nil
		}
	}
	return nil, fmt.Errorf("VM with ID %s not found", vmID)
}
