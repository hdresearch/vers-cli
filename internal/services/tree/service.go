package tree

import (
    "context"

    vers "github.com/hdresearch/vers-sdk-go"
)

// GetClusterByIdentifier fetches a cluster by ID or alias.
func GetClusterByIdentifier(ctx context.Context, client *vers.Client, idOrAlias string) (vers.APIClusterGetResponseData, error) {
    resp, err := client.API.Cluster.Get(ctx, idOrAlias)
    if err != nil {
        return vers.APIClusterGetResponseData{}, err
    }
    return resp.Data, nil
}

// GetClusterForHeadVM resolves a VM (usually HEAD) to its cluster and returns both cluster data and the VM ID.
func GetClusterForHeadVM(ctx context.Context, client *vers.Client, vmID string) (vers.APIClusterGetResponseData, error) {
    vmResp, err := client.API.Vm.Get(ctx, vmID)
    if err != nil {
        return vers.APIClusterGetResponseData{}, err
    }
    clResp, err := client.API.Cluster.Get(ctx, vmResp.Data.ClusterID)
    if err != nil {
        return vers.APIClusterGetResponseData{}, err
    }
    return clResp.Data, nil
}

