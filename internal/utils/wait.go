package utils

import (
	"context"
	"fmt"
	"time"

	vers "github.com/hdresearch/vers-sdk-go"
)

// WaitForRunning polls the VM status until it reaches the "running" state.
// Returns the final VM state or an error if the context is cancelled or the
// VM enters an unexpected terminal state.
func WaitForRunning(ctx context.Context, client *vers.Client, vmID string) error {
	const pollInterval = 2 * time.Second

	for {
		vm, err := client.Vm.Status(ctx, vmID)
		if err != nil {
			return fmt.Errorf("failed to check VM status: %w", err)
		}

		switch vm.State {
		case vers.VmStateRunning:
			return nil
		case vers.VmStateBooting:
			// Still booting, keep polling
		default:
			return fmt.Errorf("VM entered unexpected state: %s", vm.State)
		}

		select {
		case <-ctx.Done():
			return fmt.Errorf("timed out waiting for VM to start: %w", ctx.Err())
		case <-time.After(pollInterval):
		}
	}
}
