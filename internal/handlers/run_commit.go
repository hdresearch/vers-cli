package handlers

import (
	"context"
	"fmt"

	"github.com/hdresearch/vers-cli/internal/app"
	"github.com/hdresearch/vers-cli/internal/presenters"
	"github.com/hdresearch/vers-cli/internal/utils"
	vers "github.com/hdresearch/vers-sdk-go"
)

type RunCommitReq struct {
	CommitKey string
	VMAlias   string
	Wait      bool
}

func HandleRunCommit(ctx context.Context, a *app.App, r RunCommitReq) (presenters.RunCommitView, error) {
	body := vers.VmRestoreFromCommitParams{
		VmFromCommitRequest: vers.VmFromCommitRequestParam{
			CommitID: vers.F(r.CommitKey),
		},
	}

	resp, err := a.Client.Vm.RestoreFromCommit(ctx, body)
	if err != nil {
		return presenters.RunCommitView{}, err
	}

	vmID := resp.VmID

	if r.VMAlias != "" {
		_ = utils.SetAlias(r.VMAlias, vmID)
	}

	if r.Wait {
		fmt.Fprintf(a.IO.Err, "Waiting for VM %s to be running...\n", vmID)
		if err := utils.WaitForRunning(ctx, a.Client, vmID); err != nil {
			return presenters.RunCommitView{}, err
		}
	}

	return presenters.RunCommitView{RootVmID: vmID, HeadTarget: vmID, CommitKey: r.CommitKey}, nil
}
