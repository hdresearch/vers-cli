package handlers

import (
	"context"
	"fmt"

	"github.com/hdresearch/vers-cli/internal/app"
	"github.com/hdresearch/vers-cli/internal/presenters"
	"github.com/hdresearch/vers-cli/internal/utils"
)

type BranchReq struct {
	Target   string // vm id or alias; if empty, use HEAD
	Alias    string // alias for new VM
	Checkout bool   // whether to set HEAD to new VM
}

func HandleBranch(ctx context.Context, a *app.App, r BranchReq) (presenters.BranchView, error) {
	res := presenters.BranchView{}

	// Resolve source VM id
	var vmID string
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
		if err != nil {
			return res, fmt.Errorf("failed to find VM: %w", err)
		}
		vmID = vmInfo.ID
		res.FromID = vmID
		res.FromName = vmInfo.DisplayName
	}

	// Note: Alias parameter no longer supported in new SDK
	resp, err := a.Client.Vm.Branch(ctx, vmID)
	if err != nil {
		return res, fmt.Errorf("failed to create branch from vm '%s': %w", res.FromName, err)
	}

	res.NewID = resp.ID
	res.NewAlias = "" // Alias not available in new SDK
	res.NewState = "unknown" // State not available in new SDK

	if r.Checkout {
		if err := utils.SetHead(resp.ID); err != nil {
			res.CheckoutErr = err
		} else {
			res.CheckoutDone = true
		}
	}
	return res, nil
}
