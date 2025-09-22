package handlers

import (
	"context"
	"strings"

	"github.com/hdresearch/vers-cli/internal/app"
	"github.com/hdresearch/vers-cli/internal/presenters"
	histSvc "github.com/hdresearch/vers-cli/internal/services/history"
	"github.com/hdresearch/vers-cli/internal/utils"
)

type HistoryReq struct{ Target string }

func HandleHistory(ctx context.Context, a *app.App, r HistoryReq) (presenters.HistoryView, error) {
	var vmID string
	var display string
	if strings.TrimSpace(r.Target) == "" {
		head, err := utils.GetCurrentHeadVM()
		if err != nil {
			return presenters.HistoryView{}, err
		}
		vmID, display = head, head
	} else {
		vi, err := utils.ResolveVMIdentifier(ctx, a.Client, r.Target)
		if err != nil {
			return presenters.HistoryView{}, err
		}
		vmID, display = vi.ID, vi.DisplayName
	}

	// fetch VM info for alias/state if needed
	if strings.TrimSpace(display) == vmID {
		if resp, err := a.Client.API.Vm.Get(ctx, vmID); err == nil {
			vmi := utils.CreateVMInfoFromGetResponse(resp.Data)
			display = vmi.DisplayName
		}
	}

	commits, err := histSvc.GetCommits(ctx, a.Client, vmID)
	if err != nil {
		// the presenter will show a helpful message for empty/no history
		return presenters.HistoryView{VMName: display, VMID: vmID, Commits: nil}, nil
	}
	return presenters.HistoryView{VMName: display, VMID: vmID, Commits: commits}, nil
}
