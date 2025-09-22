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
	CommitKey        string
	FsSizeClusterMiB int64
	ClusterAlias     string
	VMAlias          string
}

type RunCommitView struct{ ClusterID, RootVmID, HeadTarget, CommitKey string }

func HandleRunCommit(ctx context.Context, a *app.App, r RunCommitReq) (presenters.RunCommitView, error) {
	params := vers.ClusterCreateRequestClusterFromCommitParamsParamsParam{CommitKey: vers.F(r.CommitKey)}
	if r.ClusterAlias != "" {
		params.ClusterAlias = vers.F(r.ClusterAlias)
	}
	if r.VMAlias != "" {
		params.VmAlias = vers.F(r.VMAlias)
	}
	if r.FsSizeClusterMiB > 0 {
		params.FsSizeClusterMib = vers.F(r.FsSizeClusterMiB)
	}

	body := vers.APIClusterNewParams{ClusterCreateRequest: vers.ClusterCreateRequestClusterFromCommitParamsParam{ClusterType: vers.F(vers.ClusterCreateRequestClusterFromCommitParamsClusterTypeFromCommit), Params: vers.F(params)}}
	resp, err := a.Client.API.Cluster.New(ctx, body)
	if err != nil {
		return presenters.RunCommitView{}, err
	}

	clusterInfo := resp.Data
	headTarget := clusterInfo.RootVmID
	if r.VMAlias != "" {
		headTarget = r.VMAlias
	}

	if _, err := os.Stat(".vers"); os.IsNotExist(err) {
		// warn but continue
	} else {
		headFile := filepath.Join(".vers", "HEAD")
		if err := os.WriteFile(headFile, []byte(headTarget+"\n"), 0644); err != nil {
			return presenters.RunCommitView{}, fmt.Errorf("failed to update HEAD: %w", err)
		}
	}
	return presenters.RunCommitView{ClusterID: clusterInfo.ID, RootVmID: clusterInfo.RootVmID, HeadTarget: headTarget, CommitKey: r.CommitKey}, nil
}
