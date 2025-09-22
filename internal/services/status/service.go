package status

import (
	"context"

	vers "github.com/hdresearch/vers-sdk-go"
)

func ListClusters(ctx context.Context, client *vers.Client) ([]vers.APIClusterListResponseData, error) {
	resp, err := client.API.Cluster.List(ctx)
	if err != nil {
		return nil, err
	}
	return resp.Data, nil
}

func GetCluster(ctx context.Context, client *vers.Client, idOrAlias string) (vers.APIClusterGetResponseData, error) {
	resp, err := client.API.Cluster.Get(ctx, idOrAlias)
	if err != nil {
		return vers.APIClusterGetResponseData{}, err
	}
	return resp.Data, nil
}

func GetVM(ctx context.Context, client *vers.Client, idOrAlias string) (vers.APIVmGetResponseData, error) {
	resp, err := client.API.Vm.Get(ctx, idOrAlias)
	if err != nil {
		return vers.APIVmGetResponseData{}, err
	}
	return resp.Data, nil
}
