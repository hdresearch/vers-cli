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

	// Note: Cluster concept removed, creating a new root VM instead
	// Aliases no longer supported in new SDK
	vmConfig := vers.NewRootRequestVmConfigParam{
		MemSizeMib: vers.F(r.MemSizeMib),
		VcpuCount:  vers.F(r.VcpuCount),
		FsSizeMib:  vers.F(r.FsSizeVmMib),
	}
	if r.RootfsName != "" {
		vmConfig.ImageName = vers.F(r.RootfsName)
	}
	if r.KernelName != "" {
		vmConfig.KernelName = vers.F(r.KernelName)
	}

	body := vers.VmNewRootParams{
		NewRootRequest: vers.NewRootRequestParam{
			VmConfig: vers.F(vmConfig),
		},
	}

	resp, err := a.Client.Vm.NewRoot(ctx, body)
	if err != nil {
		return presenters.RunView{}, err
	}

	vmID := resp.ID
    // Persist HEAD via utils to keep format/semantics consistent (store VM ID).
    if _, err := os.Stat(".vers"); os.IsNotExist(err) {
        // Not a vers repo; skip writing HEAD but don't fail.
    } else {
        if err := utils.SetHead(vmID); err != nil {
            return presenters.RunView{}, fmt.Errorf("failed to update HEAD: %w", err)
        }
    }

    return presenters.RunView{ClusterID: "", RootVmID: vmID, VmAlias: "", HeadTarget: vmID}, nil
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
