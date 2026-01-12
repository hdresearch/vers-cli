package handlers

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

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
	resp, err := a.Client.Vm.Branch(ctx, vmID)
	if err != nil {
		return res, fmt.Errorf("failed to create branch from vm '%s': %w", res.FromName, err)
	}

	if resp == nil {
		return res, fmt.Errorf("failed to create branch from vm '%s': empty API response", res.FromName)
	}

	newID, err := extractBranchVMID(resp.VmID, resp.JSON.RawJSON())
	if err != nil {
		return res, fmt.Errorf("failed to parse branch response: %w", err)
	}

	// New VM ID now available from Branch response in SDK alpha.24
	res.NewID = newID
	res.NewState = "unknown" // State not available in new SDK

	// Save alias locally if provided
	if r.Alias != "" {
		_ = utils.SetAlias(r.Alias, newID)
		res.NewAlias = r.Alias
	}

	if r.Checkout {
		if err := utils.SetHead(newID); err != nil {
			res.CheckoutErr = err
		} else {
			res.CheckoutDone = true
		}
	}
	return res, nil
}

func extractBranchVMID(primaryID, raw string) (string, error) {
	if primaryID != "" {
		return primaryID, nil
	}
	if strings.TrimSpace(raw) == "" {
		return "", fmt.Errorf("branch response missing vm ID")
	}

	type fallbackVM struct {
		VMID string `json:"vm_id"`
	}
	var fallback struct {
		VMID string       `json:"vm_id"`
		VMs  []fallbackVM `json:"vms"`
	}

	if err := json.Unmarshal([]byte(raw), &fallback); err != nil {
		return "", fmt.Errorf("branch response parse error: %w", err)
	}

	if fallback.VMID != "" {
		return fallback.VMID, nil
	}
	var ids []string
	for _, vm := range fallback.VMs {
		if vm.VMID != "" {
			ids = append(ids, vm.VMID)
		}
	}

	switch len(ids) {
	case 1:
		return ids[0], nil
	case 0:
		return "", fmt.Errorf("branch response missing vm ID")
	default:
		return "", fmt.Errorf("branch response contained multiple vm IDs")
	}
}
