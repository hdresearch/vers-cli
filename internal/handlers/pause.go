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

	updateParams := vers.APIVmUpdateParams{VmPatchRequest: vers.VmPatchRequestParam{State: vers.F(vers.VmPatchRequestStatePaused)}}
	resp, err := a.Client.API.Vm.Update(ctx, vmID, updateParams)
	if err != nil {
		return presenters.PauseView{}, fmt.Errorf("failed to pause VM '%s': %w", vmName, err)
	}
	return presenters.PauseView{VMName: vmName, NewState: string(resp.Data.State)}, nil
}
