package handlers

import (
	"context"
	"fmt"

	"github.com/hdresearch/vers-cli/internal/app"
	"github.com/hdresearch/vers-cli/internal/presenters"
	"github.com/hdresearch/vers-cli/internal/runconfig"
)

type BuildReq struct{ Config *runconfig.Config }

func HandleBuild(ctx context.Context, a *app.App, r BuildReq) (presenters.BuildView, error) {
	cfg := r.Config
	if cfg.Builder.Name == "none" {
		return presenters.BuildView{Skipped: true, Reason: "builder is 'none'"}, nil
	}

	// Rootfs operations have been removed from the SDK
	return presenters.BuildView{}, fmt.Errorf("custom rootfs building is no longer supported - rootfs operations have been removed from the API")
}
