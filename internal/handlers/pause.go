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

func HandlePause(ctx context.Context, a *app.App, r PauseReq) (presenters.PauseView, error) {
	resolved, err := utils.ResolveTargetVM(ctx, a.Client, r.Target)
	if err != nil {
		return presenters.PauseView{}, err
	}

	err = a.Client.Vm.UpdateState(ctx, resolved.ID, vers.VmUpdateStateParams{
		VmUpdateStateRequest: vers.VmUpdateStateRequestParam{
			State: vers.F(vers.VmUpdateStateRequestStatePaused),
		},
	})
	if err != nil {
		return presenters.PauseView{}, fmt.Errorf("failed to pause VM '%s': %w", resolved.ID, err)
	}
	return presenters.PauseView{VMName: resolved.ID, NewState: "Paused"}, nil
}
