package handlers

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"github.com/hdresearch/vers-cli/internal/app"
	"github.com/hdresearch/vers-cli/internal/presenters"
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

	vmID := resp.ID

	if _, err := os.Stat(".vers"); os.IsNotExist(err) {
		// warn but continue
	} else {
		headFile := filepath.Join(".vers", "HEAD")
		if err := os.WriteFile(headFile, []byte(vmID+"\n"), 0644); err != nil {
			return presenters.RunCommitView{}, fmt.Errorf("failed to update HEAD: %w", err)
		}
	}
	return presenters.RunCommitView{RootVmID: vmID, HeadTarget: vmID, CommitKey: r.CommitKey}, nil
}
