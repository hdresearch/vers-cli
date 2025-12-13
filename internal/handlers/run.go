package handlers

import (
	"context"

	"github.com/hdresearch/vers-cli/internal/app"
	"github.com/hdresearch/vers-cli/internal/presenters"
	"github.com/hdresearch/vers-cli/internal/utils"
	vers "github.com/hdresearch/vers-sdk-go"
)

type RunReq struct {
	MemSizeMib  int64
	VcpuCount   int64
	RootfsName  string
	KernelName  string
	FsSizeVmMib int64
	VMAlias     string
}

type RunView struct{ RootVmID, VmAlias, HeadTarget string }

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

	// SDK alpha.24 now returns the VM ID
	vmID := resp.VmID

	// Save alias locally if provided
	if r.VMAlias != "" {
		_ = utils.SetAlias(r.VMAlias, vmID)
	}

	return presenters.RunView{RootVmID: vmID, VmAlias: r.VMAlias, HeadTarget: vmID}, nil
}

func validateAndNormalize(r *RunReq) error {
	// Set default VM filesystem size if not specified
	if r.FsSizeVmMib == 0 {
		r.FsSizeVmMib = 512
	}
	return nil
}
