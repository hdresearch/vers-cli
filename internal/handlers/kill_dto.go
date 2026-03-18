package handlers

import (
	"context"
	"fmt"

	"github.com/hdresearch/vers-cli/internal/app"
	delsvc "github.com/hdresearch/vers-cli/internal/services/deletion"
	"github.com/hdresearch/vers-cli/internal/utils"
)

// KillDTO is a structured summary suitable for MCP outputs.
type KillDTO struct {
	Scope        string   `json:"scope"`
	Targets      []string `json:"targets"`
	DeletedIDs   []string `json:"deleted_ids,omitempty"`
	AffectedHead bool     `json:"affected_head,omitempty"`
}

// HandleKillDTO performs non-interactive deletion with a structured result.
func HandleKillDTO(ctx context.Context, a *app.App, r KillReq) (KillDTO, error) {
	dto := KillDTO{Targets: r.Targets, Scope: "vms"}
	targets := r.Targets
	if len(targets) == 0 {
		head, err := utils.GetCurrentHeadVM()
		if err != nil {
			return dto, fmt.Errorf("no arguments provided and %w", err)
		}
		targets = []string{head}
		dto.AffectedHead = true
	}

	for _, ref := range targets {
		vmInfo, err := utils.ResolveVMIdentifier(ctx, a.Client, ref)
		if err != nil {
			return dto, err
		}
		deletedID, err := delsvc.DeleteVM(ctx, a.Client, vmInfo.ID)
		if err != nil {
			return dto, err
		}
		dto.DeletedIDs = append(dto.DeletedIDs, deletedID)
	}
	return dto, nil
}
