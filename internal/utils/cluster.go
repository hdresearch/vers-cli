package utils

import (
	"context"
	"fmt"

	vers "github.com/hdresearch/vers-sdk-go"
)

// ClusterInfo contains both ID and display name for a cluster
type ClusterInfo struct {
	ID          string
	DisplayName string
	VmCount     int
}

// ResolveClusterIdentifier takes a cluster ID or alias and returns the cluster ID and display info
func ResolveClusterIdentifier(ctx context.Context, client *vers.Client, identifier string) (*ClusterInfo, error) {
	response, err := client.API.Cluster.Get(ctx, identifier)
	if err != nil {
		return nil, fmt.Errorf("cluster '%s' not found: %w", identifier, err)
	}

	cluster := response.Data
	displayName := cluster.Alias
	if displayName == "" {
		displayName = cluster.ID
	}

	return &ClusterInfo{
		ID:          cluster.ID,
		DisplayName: displayName,
		VmCount:     int(cluster.VmCount),
	}, nil
}

// CreateClusterInfoFromListResponse creates ClusterInfo from a List API response item
// Use this when you already have cluster data from List endpoint to avoid extra API calls
func CreateClusterInfoFromListResponse(cluster vers.APIClusterListResponseData) *ClusterInfo {
	displayName := cluster.Alias
	if displayName == "" {
		displayName = cluster.ID
	}

	return &ClusterInfo{
		ID:          cluster.ID,
		DisplayName: displayName,
		VmCount:     int(cluster.VmCount),
	}
}
