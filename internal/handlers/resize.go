package handlers

import (
	"context"
	"fmt"

	"github.com/hdresearch/vers-cli/internal/app"
	"github.com/hdresearch/vers-cli/internal/utils"
	vers "github.com/hdresearch/vers-sdk-go"
)

type ResizeReq struct {
	Target    string // vm id or alias; if empty, use HEAD
	FsSizeMib int64  // new disk size in MiB
}

func HandleResize(ctx context.Context, a *app.App, r ResizeReq) (string, error) {
	if r.FsSizeMib <= 0 {
		return "", fmt.Errorf("disk size must be a positive number (in MiB)")
	}

	vmID := r.Target
	if vmID == "" {
		var err error
		vmID, err = utils.GetCurrentHeadVM()
		if err != nil {
			return "", fmt.Errorf("no VM specified and %w", err)
		}
	} else {
		vmInfo, err := utils.ResolveVMIdentifier(ctx, a.Client, r.Target)
		if err == nil {
			vmID = vmInfo.ID
		}
		// If resolve fails, try the raw string as a VM ID
	}

	err := a.Client.Vm.ResizeDisk(ctx, vmID, vers.VmResizeDiskParams{
		VmResizeDiskRequest: vers.VmResizeDiskRequestParam{
			FsSizeMib: vers.F(r.FsSizeMib),
		},
	})
	if err != nil {
		return "", fmt.Errorf("failed to resize disk for VM '%s': %w", vmID, err)
	}

	return vmID, nil
}
