package handlers

import (
	"context"
	"fmt"

	"github.com/hdresearch/vers-cli/internal/app"
	"github.com/hdresearch/vers-cli/internal/presenters"
	"github.com/hdresearch/vers-cli/internal/utils"
)

type InfoReq struct {
	Target string // vm id or alias; if empty, use HEAD
}

func HandleInfo(ctx context.Context, a *app.App, r InfoReq) (presenters.InfoView, error) {
	vmID := r.Target
	if vmID == "" {
		var err error
		vmID, err = utils.GetCurrentHeadVM()
		if err != nil {
			return presenters.InfoView{}, fmt.Errorf("no VM specified and %w", err)
		}
		return handleInfoByID(ctx, a, vmID, true)
	}

	// Try to resolve as alias first
	vmInfo, err := utils.ResolveVMIdentifier(ctx, a.Client, r.Target)
	if err != nil {
		// Could be a raw VM ID that just isn't in the list — try metadata directly
		return handleInfoByID(ctx, a, r.Target, false)
	}
	return handleInfoByID(ctx, a, vmInfo.ID, false)
}

func handleInfoByID(ctx context.Context, a *app.App, vmID string, usedHead bool) (presenters.InfoView, error) {
	meta, err := a.Client.Vm.GetMetadata(ctx, vmID)
	if err != nil {
		return presenters.InfoView{}, fmt.Errorf("failed to get metadata for VM '%s': %w", vmID, err)
	}

	return presenters.InfoView{
		Metadata: meta,
		UsedHEAD: usedHead,
	}, nil
}
