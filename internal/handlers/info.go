package handlers

import (
	"context"
	"fmt"

	"github.com/hdresearch/vers-cli/internal/app"
	"github.com/hdresearch/vers-cli/internal/presenters"
	"github.com/hdresearch/vers-cli/internal/utils"
)

type InfoReq struct {
	Target string
}

func HandleInfo(ctx context.Context, a *app.App, r InfoReq) (presenters.InfoView, error) {
	t, err := utils.ResolveTarget(r.Target)
	if err != nil {
		return presenters.InfoView{}, err
	}

	// Resolve alias if needed, but don't fail if it's a raw ID
	vmID := utils.ResolveAlias(t.Ident)

	meta, err := a.Client.Vm.GetMetadata(ctx, vmID)
	if err != nil {
		return presenters.InfoView{}, fmt.Errorf("failed to get metadata for VM '%s': %w", vmID, err)
	}

	return presenters.InfoView{
		Metadata: meta,
		UsedHEAD: t.UsedHEAD,
	}, nil
}
