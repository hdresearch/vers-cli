package handlers

import (
	"context"
	"fmt"

	"github.com/hdresearch/vers-cli/internal/app"
	"github.com/hdresearch/vers-cli/internal/presenters"
	"github.com/hdresearch/vers-cli/internal/utils"
	vers "github.com/hdresearch/vers-sdk-go"
)

type ResumeReq struct {
	Target string
	Wait   bool
}

func HandleResume(ctx context.Context, a *app.App, r ResumeReq) (presenters.ResumeView, error) {
	resolved, err := utils.ResolveTargetVM(ctx, a.Client, r.Target)
	if err != nil {
		return presenters.ResumeView{}, err
	}

	err = a.Client.Vm.UpdateState(ctx, resolved.ID, vers.VmUpdateStateParams{
		VmUpdateStateRequest: vers.VmUpdateStateRequestParam{
			State: vers.F(vers.VmUpdateStateRequestStateRunning),
		},
	})
	if err != nil {
		return presenters.ResumeView{}, fmt.Errorf("failed to resume VM '%s': %w", resolved.ID, err)
	}

	if r.Wait {
		fmt.Fprintf(a.IO.Err, "Waiting for VM %s to be running...\n", resolved.ID)
		if err := utils.WaitForRunning(ctx, a.Client, resolved.ID); err != nil {
			return presenters.ResumeView{}, err
		}
	}

	return presenters.ResumeView{VMName: resolved.ID, NewState: "Running"}, nil
}
