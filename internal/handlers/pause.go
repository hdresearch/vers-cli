package handlers

import (
	"context"
	"fmt"

	"github.com/hdresearch/vers-cli/internal/app"
	"github.com/hdresearch/vers-cli/internal/presenters"
	"github.com/hdresearch/vers-cli/internal/utils"
	vers "github.com/hdresearch/vers-sdk-go"
)

type PauseReq struct{ Target string }

type PauseView struct{ VMName, NewState string }

func HandlePause(ctx context.Context, a *app.App, r PauseReq) (presenters.PauseView, error) {
	var vmID string
	var vmName string

	if r.Target == "" {
		head, err := utils.GetCurrentHeadVM()
		if err != nil {
			return presenters.PauseView{}, fmt.Errorf("no VM ID provided and %w", err)
		}
		vmID = head
		vmName = head
	} else {
		info, err := utils.ResolveVMIdentifier(ctx, a.Client, r.Target)
		if err != nil {
			return presenters.PauseView{}, fmt.Errorf("failed to find VM: %w", err)
		}
		vmID = info.ID
		vmName = info.DisplayName
	}

	updateParams := vers.VmUpdateStateParams{
		VmUpdateStateRequest: vers.VmUpdateStateRequestParam{
			State: vers.F(vers.VmUpdateStateRequestStatePaused),
		},
	}
	err := a.Client.Vm.UpdateState(ctx, vmID, updateParams)
	if err != nil {
		return presenters.PauseView{}, fmt.Errorf("failed to pause VM '%s': %w", vmName, err)
	}
	return presenters.PauseView{VMName: vmName, NewState: "Paused"}, nil
}
