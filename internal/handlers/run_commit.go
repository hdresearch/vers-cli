package handlers

import (
	"context"

	"github.com/hdresearch/vers-cli/internal/app"
	"github.com/hdresearch/vers-cli/internal/presenters"
	"github.com/hdresearch/vers-cli/internal/utils"
	vers "github.com/hdresearch/vers-sdk-go"
)

type RunCommitReq struct {
	CommitKey string
	VMAlias   string
}

type RunCommitView struct{ RootVmID, HeadTarget, CommitKey string }

func HandleRunCommit(ctx context.Context, a *app.App, r RunCommitReq) (presenters.RunCommitView, error) {
	// Note: Cluster concept removed, creating VM from commit instead
	// CommitKey is now CommitID (UUID) in new SDK
	body := vers.VmRestoreFromCommitParams{
		VmFromCommitRequest: vers.VmFromCommitRequestParam{
			CommitID: vers.F(r.CommitKey),
		},
	}

	resp, err := a.Client.Vm.RestoreFromCommit(ctx, body)
	if err != nil {
		return presenters.RunCommitView{}, err
	}

	// SDK alpha.24 now returns the VM ID
	vmID := resp.VmID

	// Save alias locally if provided
	if r.VMAlias != "" {
		_ = utils.SetAlias(r.VMAlias, vmID)
	}

	return presenters.RunCommitView{RootVmID: vmID, HeadTarget: vmID, CommitKey: r.CommitKey}, nil
}
