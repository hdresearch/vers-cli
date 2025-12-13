package deletion

import (
	"context"
	"strings"

	"github.com/hdresearch/vers-cli/internal/errorsx"
	vers "github.com/hdresearch/vers-sdk-go"
)

// DeleteVM deletes a VM by ID. If recursive is false and the VM has children, returns *errorsx.HasChildrenError.
// If the VM is a root VM, returns *errorsx.IsRootError.
func DeleteVM(ctx context.Context, client *vers.Client, vmID string, recursive bool) ([]string, error) {
	result, err := client.Vm.Delete(ctx, vmID, vers.VmDeleteParams{})
	if err != nil {
		es := err.Error()
		if strings.Contains(es, "HasChildren") {
			return nil, &errorsx.HasChildrenError{VMID: vmID}
		}
		if strings.Contains(es, "IsRoot") {
			return nil, &errorsx.IsRootError{VMID: vmID}
		}
		return nil, err
	}
	return []string{result.VmID}, nil
}
