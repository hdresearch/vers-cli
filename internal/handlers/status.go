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
	Cluster string // --cluster value; if set, show cluster
	Target  string // optional VM identifier; if set, show VM
}

// HandleStatus performs the status command logic using services and utilities.
func HandleStatus(ctx context.Context, a *app.App, req StatusReq) (presenters.StatusView, error) {
	res := presenters.StatusView{}

	// Show head status only when neither cluster nor target requested
	res.Head.Show = (req.Cluster == "" && req.Target == "")
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
            // Brief VM info fetch with short timeout, but don't fail the command if it errors.
            hctx, cancel := context.WithTimeout(ctx, minDuration(a.Timeouts.APIShort, 3*time.Second))
            defer cancel()
            if vmResp, err := a.Client.API.Vm.Get(hctx, headID); err == nil {
                vmInfo := utils.CreateVMInfoFromGetResponse(vmResp.Data)
                res.Head.DisplayName = vmInfo.DisplayName
                res.Head.State = vmInfo.State
            }
        }
    }

	switch {
	case req.Cluster != "":
		cl, err := svc.GetCluster(ctx, a.Client, req.Cluster)
		if err != nil {
			return res, err
		}
		res.Mode = presenters.StatusCluster
		res.Cluster = cl
		return res, nil
	case req.Target != "":
		vm, err := svc.GetVM(ctx, a.Client, req.Target)
		if err != nil {
			return res, err
		}
		res.Mode = presenters.StatusVM
		res.VM = vm
		return res, nil
	default:
		list, err := svc.ListClusters(ctx, a.Client)
		if err != nil {
			return res, err
		}
		res.Mode = presenters.StatusList
		res.Clusters = list
		return res, nil
	}
}

func minDuration(a, b time.Duration) time.Duration {
	if a < b {
		return a
	}
	return b
}
