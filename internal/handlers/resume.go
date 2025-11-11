package handlers

import (
	"context"
	"fmt"

	"github.com/hdresearch/vers-cli/internal/app"
	"github.com/hdresearch/vers-cli/internal/presenters"
	"github.com/hdresearch/vers-cli/internal/utils"
	vers "github.com/hdresearch/vers-sdk-go"
)

type ResumeReq struct{ Target string }

func HandleResume(ctx context.Context, a *app.App, r ResumeReq) (presenters.ResumeView, error) {
	var vmID string
	var vmName string
	if r.Target == "" {
		head, err := utils.GetCurrentHeadVM()
		if err != nil {
			return presenters.ResumeView{}, fmt.Errorf("no VM ID provided and %w", err)
		}
		vmID, vmName = head, head
	} else {
		info, err := utils.ResolveVMIdentifier(ctx, a.Client, r.Target)
		if err != nil {
			return presenters.ResumeView{}, fmt.Errorf("failed to find VM: %w", err)
		}
		vmID, vmName = info.ID, info.DisplayName
	}

	updateParams := vers.VmUpdateStateParams{
		VmUpdateStateRequest: vers.VmUpdateStateRequestParam{
			State: vers.F(vers.VmUpdateStateRequestStateRunning),
		},
	}
	err := a.Client.Vm.UpdateState(ctx, vmID, updateParams)
	if err != nil {
		return presenters.ResumeView{}, fmt.Errorf("failed to resume VM '%s': %w", vmName, err)
	}
	return presenters.ResumeView{VMName: vmName, NewState: "Running"}, nil
}
