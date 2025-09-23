package handlers

import (
	"context"
	"fmt"
	"os"

	"github.com/hdresearch/vers-cli/internal/app"
	"github.com/hdresearch/vers-cli/internal/presenters"
	"github.com/hdresearch/vers-cli/internal/utils"
	vers "github.com/hdresearch/vers-sdk-go"
)

type RunReq struct {
	MemSizeMib       int64
	VcpuCount        int64
	RootfsName       string
	KernelName       string
	FsSizeClusterMib int64
	FsSizeVmMib      int64
	ClusterAlias     string
	VMAlias          string
}

type RunView struct{ ClusterID, RootVmID, VmAlias, HeadTarget string }

func HandleRun(ctx context.Context, a *app.App, r RunReq) (presenters.RunView, error) {
	if err := validateAndNormalize(&r); err != nil {
		return presenters.RunView{}, err
	}

	params := vers.ClusterCreateRequestNewClusterParamsParamsParam{
		MemSizeMib:       vers.F(r.MemSizeMib),
		VcpuCount:        vers.F(r.VcpuCount),
		RootfsName:       vers.F(r.RootfsName),
		KernelName:       vers.F(r.KernelName),
		FsSizeClusterMib: vers.F(r.FsSizeClusterMib),
		FsSizeVmMib:      vers.F(r.FsSizeVmMib),
	}
	if r.ClusterAlias != "" {
		params.ClusterAlias = vers.F(r.ClusterAlias)
	}
	if r.VMAlias != "" {
		params.VmAlias = vers.F(r.VMAlias)
	}

	body := vers.APIClusterNewParams{ClusterCreateRequest: vers.ClusterCreateRequestNewClusterParamsParam{ClusterType: vers.F(vers.ClusterCreateRequestNewClusterParamsClusterTypeNew), Params: vers.F(params)}}
	resp, err := a.Client.API.Cluster.New(ctx, body)
	if err != nil {
		return presenters.RunView{}, err
	}

	clusterInfo := resp.Data
	// Persist HEAD via utils to keep format/semantics consistent (store VM ID).
	// Prefer storing the new root VM ID; if alias was provided, still store ID
	// but show the alias in the presenter for UX.
	if _, err := os.Stat(".vers"); os.IsNotExist(err) {
		// Not a vers repo; skip writing HEAD but don't fail.
	} else {
		if err := utils.SetHead(clusterInfo.RootVmID); err != nil {
			return presenters.RunView{}, fmt.Errorf("failed to update HEAD: %w", err)
		}
	}
	headDisplay := r.VMAlias
	if headDisplay == "" {
		headDisplay = clusterInfo.RootVmID
	}
	return presenters.RunView{ClusterID: clusterInfo.ID, RootVmID: clusterInfo.RootVmID, VmAlias: r.VMAlias, HeadTarget: headDisplay}, nil
}

func validateAndNormalize(r *RunReq) error {
	if r.FsSizeClusterMib == 0 && r.FsSizeVmMib == 0 {
		r.FsSizeClusterMib = 1024
		r.FsSizeVmMib = 512
	} else if r.FsSizeClusterMib == 0 && r.FsSizeVmMib > 0 {
		vm := r.FsSizeVmMib
		cluster := vm * 2
		if cluster < 1024 {
			cluster = 1024
		}
		r.FsSizeClusterMib = cluster
	} else if r.FsSizeVmMib == 0 && r.FsSizeClusterMib > 0 {
		r.FsSizeVmMib = r.FsSizeClusterMib / 2
	}
	if r.FsSizeVmMib > r.FsSizeClusterMib {
		return fmt.Errorf("invalid configuration: VM filesystem size (%d MiB) must not exceed cluster filesystem size (%d MiB). Use --fs-size-cluster and --fs-size-vm or update vers.toml", r.FsSizeVmMib, r.FsSizeClusterMib)
	}
	return nil
}
