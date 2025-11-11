package handlers

import (
    "context"
    "errors"
    "time"

	"github.com/hdresearch/vers-cli/internal/app"
	"github.com/hdresearch/vers-cli/internal/presenters"
	svc "github.com/hdresearch/vers-cli/internal/services/status"
	"github.com/hdresearch/vers-cli/internal/utils"
)

// StatusReq captures the parsed flags/args from the status command.
type StatusReq struct {
	Target string // optional VM identifier; if set, show VM
}

// HandleStatus performs the status command logic using services and utilities.
func HandleStatus(ctx context.Context, a *app.App, req StatusReq) (presenters.StatusView, error) {
	res := presenters.StatusView{}

	// Show head status only when target not requested
	res.Head.Show = (req.Target == "")
    if res.Head.Show {
        headID, err := utils.GetCurrentHeadVM()
        if err != nil {
            // HEAD missing or empty
            res.Head.Present = false
            if errors.Is(err, utils.ErrHeadEmpty) {
                res.Head.Empty = true
            }
        } else {
            res.Head.Present = true
            res.Head.ID = headID
            res.Head.DisplayName = headID
            // Note: State/Alias no longer available in new SDK
        }
    }

	if req.Target != "" {
		vm, err := svc.GetVM(ctx, a.Client, req.Target)
		if err != nil {
			return res, err
		}
		res.Mode = presenters.StatusVM
		res.VM = vm
		return res, nil
	}

	list, err := svc.ListVMs(ctx, a.Client)
	if err != nil {
		return res, err
	}
	res.Mode = presenters.StatusList
	res.VMs = list
	return res, nil
}

func minDuration(a, b time.Duration) time.Duration {
	if a < b {
		return a
	}
	return b
}
