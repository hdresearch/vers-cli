package handlers

import (
	"context"
	"fmt"

	"github.com/hdresearch/vers-cli/internal/app"
	"github.com/hdresearch/vers-cli/internal/utils"
	vers "github.com/hdresearch/vers-sdk-go"
)

type ResizeReq struct {
	Target    string
	FsSizeMib int64
}

func HandleResize(ctx context.Context, a *app.App, r ResizeReq) (string, error) {
	if r.FsSizeMib <= 0 {
		return "", fmt.Errorf("disk size must be a positive number (in MiB)")
	}

	resolved, err := utils.ResolveTargetVM(ctx, a.Client, r.Target)
	if err != nil {
		return "", err
	}

	err = a.Client.Vm.ResizeDisk(ctx, resolved.ID, vers.VmResizeDiskParams{
		VmResizeDiskRequest: vers.VmResizeDiskRequestParam{
			FsSizeMib: vers.F(r.FsSizeMib),
		},
	})
	if err != nil {
		return "", fmt.Errorf("failed to resize disk for VM '%s': %w", resolved.ID, err)
	}

	return resolved.ID, nil
}
