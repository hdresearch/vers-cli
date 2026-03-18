package handlers

import (
	"context"
	"fmt"

	"github.com/hdresearch/vers-cli/internal/app"
	"github.com/hdresearch/vers-cli/internal/utils"
	vers "github.com/hdresearch/vers-sdk-go"
)

// CommitCreateReq is the request for creating a commit.
type CommitCreateReq struct {
	Target string // vm id or alias; if empty, use HEAD
}

// CommitCreateView is the result of creating a commit.
type CommitCreateView struct {
	CommitID string `json:"commit_id"`
	VmID     string `json:"vm_id"`
	UsedHEAD bool   `json:"used_head,omitempty"`
}

func HandleCommitCreate(ctx context.Context, a *app.App, r CommitCreateReq) (CommitCreateView, error) {
	var vmID string
	var usedHead bool

	if r.Target != "" {
		info, err := utils.ResolveVMIdentifier(ctx, a.Client, r.Target)
		if err != nil {
			return CommitCreateView{}, fmt.Errorf("failed to find VM: %w", err)
		}
		vmID = info.ID
	} else {
		var err error
		vmID, err = utils.GetCurrentHeadVM()
		if err != nil {
			return CommitCreateView{}, fmt.Errorf("failed to get current VM: %w", err)
		}
		usedHead = true
	}

	resp, err := a.Client.Vm.Commit(ctx, vmID, vers.VmCommitParams{})
	if err != nil {
		return CommitCreateView{}, fmt.Errorf("failed to commit VM '%s': %w", vmID, err)
	}

	return CommitCreateView{
		CommitID: resp.CommitID,
		VmID:     vmID,
		UsedHEAD: usedHead,
	}, nil
}
