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
	Wait     bool   // block until new VMs are running
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

	// Resolve source VM (falls back to HEAD if empty)
	t, err := utils.ResolveTarget(r.Target)
	if err != nil {
		return res, err
	}
	res.UsedHEAD = t.UsedHEAD
	vmID := t.Ident

	// Try to verify the VM exists; if resolve fails, it may be a commit ID
	if info, err := utils.ResolveVMIdentifier(ctx, a.Client, t.Ident); err == nil {
		vmID = info.ID
	}
	res.FromID = vmID
	res.FromName = vmID

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

	if r.Wait {
		fmt.Fprintf(a.IO.Err, "Waiting for %d VM(s) to be running...\n", len(res.NewIDs))
		for _, id := range res.NewIDs {
			if err := utils.WaitForRunning(ctx, a.Client, id); err != nil {
				return res, fmt.Errorf("wait failed for VM %s: %w", id, err)
			}
		}
	}

	return res, nil
}
