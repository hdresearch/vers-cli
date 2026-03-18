package deletion

import (
	"context"

	vers "github.com/hdresearch/vers-sdk-go"
)

// DeleteVM deletes a VM by ID. Returns the deleted VM ID.
func DeleteVM(ctx context.Context, client *vers.Client, vmID string) (string, error) {
	result, err := client.Vm.Delete(ctx, vmID, vers.VmDeleteParams{})
	if err != nil {
		return "", err
	}
	return result.VmID, nil
}
