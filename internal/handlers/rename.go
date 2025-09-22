package handlers

import (
	"context"
	"fmt"

	"github.com/hdresearch/vers-cli/internal/app"
	"github.com/hdresearch/vers-cli/internal/presenters"
	"github.com/hdresearch/vers-cli/internal/utils"
	vers "github.com/hdresearch/vers-sdk-go"
)

type RenameReq struct {
	IsCluster bool
	Target    string
	NewAlias  string
}

func HandleRename(ctx context.Context, a *app.App, r RenameReq) (presenters.RenameView, error) {
	if r.IsCluster {
		if r.Target == "" {
			return presenters.RenameView{}, fmt.Errorf("cluster ID or alias must be provided when renaming clusters")
		}
		ci, err := utils.ResolveClusterIdentifier(ctx, a.Client, r.Target)
		if err != nil {
			return presenters.RenameView{}, fmt.Errorf("failed to find cluster: %w", err)
		}
		params := vers.APIClusterUpdateParams{ClusterPatchRequest: vers.ClusterPatchRequestParam{Alias: vers.F(r.NewAlias)}}
		resp, err := a.Client.API.Cluster.Update(ctx, ci.ID, params)
		if err != nil {
			return presenters.RenameView{}, fmt.Errorf("failed to rename cluster '%s': %w", ci.DisplayName, err)
		}
		return presenters.RenameView{Kind: "cluster", ID: resp.Data.ID, Alias: resp.Data.Alias}, nil
	}

	// VM rename
	var vmID, display string
	if r.Target == "" {
		head, err := utils.GetCurrentHeadVM()
		if err != nil {
			return presenters.RenameView{}, fmt.Errorf("no ID provided and %w", err)
		}
		vmID = head
		display = head
	} else {
		vi, err := utils.ResolveVMIdentifier(ctx, a.Client, r.Target)
		if err != nil {
			return presenters.RenameView{}, fmt.Errorf("failed to find VM: %w", err)
		}
		vmID = vi.ID
		display = vi.DisplayName
	}
	params := vers.APIVmUpdateParams{VmPatchRequest: vers.VmPatchRequestParam{Alias: vers.F(r.NewAlias)}}
	resp, err := a.Client.API.Vm.Update(ctx, vmID, params)
	if err != nil {
		return presenters.RenameView{}, fmt.Errorf("failed to rename VM '%s': %w", display, err)
	}
	return presenters.RenameView{Kind: "vm", ID: resp.Data.ID, Alias: resp.Data.Alias}, nil
}
