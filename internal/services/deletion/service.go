package deletion

import (
	"context"
	"fmt"
	"strings"

	"github.com/hdresearch/vers-cli/internal/errorsx"
	vers "github.com/hdresearch/vers-sdk-go"
)

// DeleteVM deletes a VM by ID. If recursive is false and the VM has children, returns *errorsx.HasChildrenError.
// If the VM is the root of a cluster, returns *errorsx.IsRootError.
func DeleteVM(ctx context.Context, client *vers.Client, vmID string, recursive bool) ([]string, error) {
	params := vers.APIVmDeleteParams{Recursive: vers.F(recursive)}
	result, err := client.API.Vm.Delete(ctx, vmID, params)
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
	if len(result.Data.Errors) > 0 {
		return nil, fmt.Errorf("deletion had errors")
	}
	return result.Data.DeletedIDs, nil
}

// DeleteCluster deletes a cluster by ID and returns deleted VM IDs.
func DeleteCluster(ctx context.Context, client *vers.Client, clusterID string) ([]string, error) {
	result, err := client.API.Cluster.Delete(ctx, clusterID)
	if err != nil {
		return nil, err
	}
	// Inline summary detection to avoid presenter deps
	hasErrors := len(result.Data.Vms.Errors) > 0 || result.Data.FsError != ""
	if hasErrors {
		summary := result.Data.FsError
		for _, vmErr := range result.Data.Vms.Errors {
			if summary != "" {
				summary += "; "
			}
			summary += fmt.Sprintf("%s: %s", vmErr.ID, vmErr.Error)
		}
		return nil, fmt.Errorf("partially failed: %s", summary)
	}
	return result.Data.Vms.DeletedIDs, nil
}
