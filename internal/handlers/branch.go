package handlers

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"

	"github.com/hdresearch/vers-cli/internal/app"
	"github.com/hdresearch/vers-cli/internal/presenters"
	"github.com/hdresearch/vers-cli/internal/utils"
	"github.com/hdresearch/vers-sdk-go/option"
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
	var opts []option.RequestOption
	if count > 1 {
		opts = append(opts, option.WithQuery("count", strconv.Itoa(count)))
	}
	resp, err := a.Client.Vm.Branch(ctx, vmID, opts...)
	if err != nil {
		return res, fmt.Errorf("failed to create branch from vm '%s': %w", res.FromName, err)
	}

	if resp == nil {
		return res, fmt.Errorf("failed to create branch from vm '%s': empty API response", res.FromName)
	}

	newIDs, err := extractBranchVMIDs(resp.VmID, resp.JSON.RawJSON())
	if err != nil {
		return res, fmt.Errorf("failed to parse branch response: %w", err)
	}
	if len(newIDs) == 0 {
		return res, fmt.Errorf("failed to parse branch response: no VM IDs returned")
	}
	if count > 1 && len(newIDs) != count {
		return res, fmt.Errorf("expected %d new VMs, but API returned %d", count, len(newIDs))
	}

	// New VM IDs now available from Branch response in SDK alpha.24
	res.NewIDs = newIDs
	res.NewID = newIDs[0]
	res.NewState = "unknown" // State not available in new SDK

	// Save alias locally if provided
	if r.Alias != "" {
		_ = utils.SetAlias(r.Alias, newIDs[0])
		res.NewAlias = r.Alias
	}

	if r.Checkout {
		if err := utils.SetHead(newIDs[0]); err != nil {
			res.CheckoutErr = err
		} else {
			res.CheckoutDone = true
		}
	}
	return res, nil
}

func extractBranchVMIDs(primaryID, raw string) ([]string, error) {
	if primaryID != "" {
		return []string{primaryID}, nil
	}
	if strings.TrimSpace(raw) == "" {
		return nil, fmt.Errorf("branch response missing vm ID")
	}

	type fallbackVM struct {
		VMID string `json:"vm_id"`
	}
	var fallback struct {
		VMID string       `json:"vm_id"`
		VMs  []fallbackVM `json:"vms"`
	}

	if err := json.Unmarshal([]byte(raw), &fallback); err != nil {
		return nil, fmt.Errorf("branch response parse error: %w", err)
	}

	var ids []string
	if fallback.VMID != "" {
		ids = append(ids, fallback.VMID)
	}
	for _, vm := range fallback.VMs {
		if vm.VMID != "" {
			ids = append(ids, vm.VMID)
		}
	}
	if len(ids) == 0 {
		return nil, fmt.Errorf("branch response missing vm ID")
	}
	return ids, nil
}
