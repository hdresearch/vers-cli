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
	Scope        string              `json:"scope"` // "vms"
	Targets      []string            `json:"targets"`
	Recursive    bool                `json:"recursive"`
	DeletedIDs   []string            `json:"deletedIds,omitempty"`
	DeletedByRef map[string][]string `json:"deletedByRef,omitempty"` // input target -> deleted ids
	AffectedHead bool                `json:"affectedHead,omitempty"`
}

// HandleKillDTO performs non-interactive deletion with a structured result.
// It does not prompt and does not print; callers must enforce confirmation policy.
func HandleKillDTO(ctx context.Context, a *app.App, r KillReq) (KillDTO, error) {
	dto := KillDTO{Targets: r.Targets, Recursive: r.Recursive, Scope: "vms"}
	targets := r.Targets
	if len(targets) == 0 {
		head, err := utils.GetCurrentHeadVM()
		if err != nil {
			return dto, fmt.Errorf("no arguments provided and %w", err)
		}
		targets = []string{head}
		dto.AffectedHead = true
	}

	dto.DeletedByRef = map[string][]string{}
	for _, ref := range targets {
		vmInfo, err := utils.ResolveVMIdentifier(ctx, a.Client, ref)
		if err != nil {
			return dto, err
		}
		ids, err := delsvc.DeleteVM(ctx, a.Client, vmInfo.ID, r.Recursive)
		if err != nil {
			return dto, err
		}
		dto.DeletedIDs = append(dto.DeletedIDs, ids...)
		dto.DeletedByRef[ref] = ids
	}
	return dto, nil
}
