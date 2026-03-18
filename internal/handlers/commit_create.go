package handlers

import (
	"context"
	"fmt"

	"github.com/hdresearch/vers-cli/internal/app"
	"github.com/hdresearch/vers-cli/internal/utils"
	vers "github.com/hdresearch/vers-sdk-go"
)

type CommitCreateReq struct {
	Target string
}

type CommitCreateView struct {
	CommitID string `json:"commit_id"`
	VmID     string `json:"vm_id"`
	UsedHEAD bool   `json:"used_head,omitempty"`
}

func HandleCommitCreate(ctx context.Context, a *app.App, r CommitCreateReq) (CommitCreateView, error) {
	resolved, err := utils.ResolveTargetVM(ctx, a.Client, r.Target)
	if err != nil {
		return CommitCreateView{}, err
	}

	resp, err := a.Client.Vm.Commit(ctx, resolved.ID, vers.VmCommitParams{})
	if err != nil {
		return CommitCreateView{}, fmt.Errorf("failed to commit VM '%s': %w", resolved.ID, err)
	}

	return CommitCreateView{
		CommitID: resp.CommitID,
		VmID:     resolved.ID,
		UsedHEAD: resolved.UsedHEAD,
	}, nil
}
