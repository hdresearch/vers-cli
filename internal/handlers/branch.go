package handlers

import (
	"context"
	"fmt"

	"github.com/hdresearch/vers-cli/internal/app"
	"github.com/hdresearch/vers-cli/internal/presenters"
	"github.com/hdresearch/vers-cli/internal/utils"
	"github.com/hdresearch/vers-sdk-go"
)

type BranchReq struct {
	Target   string // vm id or alias; if empty, use HEAD
	Alias    string // alias for new VM
	Checkout bool   // whether to set HEAD to new VM
	Count    int    // number of branches to create
}

func HandleBranch(ctx context.Context, a *app.App, r BranchReq) (presenters.BranchView, error) {
	res := presenters.BranchView{}
	count := r.Count
	if count == 0 {
		count = 1
	}
	if count < 1 {
		return res, fmt.Errorf("count must be at least 1")
	}
	if count > 1 {
		if r.Alias != "" {
			return res, fmt.Errorf("cannot set alias when creating multiple branches")
		}
	}

	// Resolve source VM id
	vmID := r.Target
	var vmInfo *utils.VMInfo
	var err error
	if r.Target == "" {
		vmID, err = utils.GetCurrentHeadVM()
		if err != nil {
			return res, fmt.Errorf("no VM ID provided and %w", err)
		}
		res.UsedHEAD = true
		res.FromID = vmID
		res.FromName = vmID
	} else {
		vmInfo, err = utils.ResolveVMIdentifier(ctx, a.Client, r.Target)
		// If VM cannot be resolved, this request may be targeting a commit ID instead
		if err == nil {
			vmID = vmInfo.ID
			res.FromID = vmID
			res.FromName = vmInfo.DisplayName
		}
	}

	// Note: Alias parameter no longer supported in new SDK
	// SDK alpha.24 now returns the new VM ID
	resp, err := a.Client.Vm.Branch(ctx, vmID, vers.VmBranchParams{Count: vers.F(int64(count))})
	if err != nil {
		return res, fmt.Errorf("failed to create branch from vm '%s': %w", res.FromName, err)
	}
	if resp == nil {
		return res, fmt.Errorf("failed to create branch from vm '%s': empty API response", res.FromName)
	}
	if len(resp.Vms) != count {
		return res, fmt.Errorf("expected %d new VMs, but API returned %d", count, len(resp.Vms))
	}

	for _, vm := range resp.Vms {
		res.NewIDs = append(res.NewIDs, vm.VmID)
	}

	// Save alias locally if provided
	if r.Alias != "" {
		_ = utils.SetAlias(r.Alias, resp.Vms[0].VmID)
		res.NewAlias = r.Alias
	}

	if r.Checkout {
		if err := utils.SetHead(resp.Vms[0].VmID); err != nil {
			res.CheckoutErr = err
		} else {
			res.CheckoutDone = true
		}
	}
	return res, nil
}
