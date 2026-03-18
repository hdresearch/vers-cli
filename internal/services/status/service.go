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

// GetVM retrieves a single VM by ID using the status endpoint.
func GetVM(ctx context.Context, client *vers.Client, vmID string) (*vers.Vm, error) {
	resp, err := client.Vm.Status(ctx, vmID)
	if err != nil {
		return nil, fmt.Errorf("VM with ID %s not found: %w", vmID, err)
	}
	return resp, nil
}
