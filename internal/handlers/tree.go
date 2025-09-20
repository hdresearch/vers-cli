package handlers

import (
	"context"

	"github.com/hdresearch/vers-cli/internal/app"
	svc "github.com/hdresearch/vers-cli/internal/services/tree"
	"github.com/hdresearch/vers-cli/internal/utils"
)

type TreeReq struct{ ClusterIdentifier string }

type TreeView struct {
	Cluster  presentersCluster
	HeadVMID string
}

// presentersCluster is a tiny alias to avoid importing SDK in the handler’s public API; the presenter already depends on it.
type presentersCluster = struct { /* filled via type alias below */
}

// We’ll just pass through the actual type via a private field and unwrap in presenter controller.
type treeInternal struct{ Cluster any }

func HandleTree(ctx context.Context, a *app.App, r TreeReq) (any, string, error) {
	var head string
	if r.ClusterIdentifier == "" {
		vm, err := utils.GetCurrentHeadVM()
		if err != nil {
			return nil, "", err
		}
		head = vm
		cluster, err := svc.GetClusterForHeadVM(ctx, a.Client, vm)
		if err != nil {
			return nil, "", err
		}
		return cluster, head, nil
	}
	cluster, err := svc.GetClusterByIdentifier(ctx, a.Client, r.ClusterIdentifier)
	if err != nil {
		return nil, "", err
	}
	// Try read head (optional)
	if vm, err := utils.GetCurrentHeadVM(); err == nil {
		head = vm
	}
	return cluster, head, nil
}
