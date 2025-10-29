package handlers

import (
	"context"

	"github.com/hdresearch/vers-cli/internal/app"
	"github.com/hdresearch/vers-cli/internal/utils"
)

type TreeReq struct{}

func HandleTree(ctx context.Context, a *app.App, r TreeReq) (any, string, error) {
	// Tree functionality requires cluster concept which has been removed from the API
	// For now, return an error indicating this feature is not available
	// TODO: Implement new tree visualization using VM parent relationships
	var head string
	if vm, err := utils.GetCurrentHeadVM(); err == nil {
		head = vm
	}
	// Return empty list of VMs for now - tree presenter will handle deprecation message
	return []struct{}{}, head, nil
}
