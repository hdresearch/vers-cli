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
	// CommitKey is either a commit ID (UUID) or, when IsRef is true, a
	// repository reference in "repo_name:tag_name" format.
	CommitKey string
	VMAlias   string
	Wait      bool
	// IsRef switches the underlying API payload from {"commit_id": ...} to
	// {"ref": ...}, which enables resolving own-org repository tags like
	// "my-app:latest" instead of raw commit UUIDs.
	IsRef bool
}

func HandleRunCommit(ctx context.Context, a *app.App, r RunCommitReq) (presenters.RunCommitView, error) {
	var reqUnion vers.VmFromCommitRequestUnionParam
	if r.IsRef {
		reqUnion = vers.VmFromCommitRequestRefParam{
			Ref: vers.F(r.CommitKey),
		}
	} else {
		reqUnion = vers.VmFromCommitRequestCommitIDParam{
			CommitID: vers.F(r.CommitKey),
		}
	}
	body := vers.VmRestoreFromCommitParams{
		VmFromCommitRequest: reqUnion,
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
