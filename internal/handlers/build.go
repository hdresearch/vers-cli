package handlers

import (
	"context"
	"fmt"
	"os"

	"github.com/hdresearch/vers-cli/internal/app"
	"github.com/hdresearch/vers-cli/internal/presenters"
	"github.com/hdresearch/vers-cli/internal/runconfig"
	buildsvc "github.com/hdresearch/vers-cli/internal/services/build"
	rootfssvc "github.com/hdresearch/vers-cli/internal/services/rootfs"
)

type BuildReq struct{ Config *runconfig.Config }

func HandleBuild(ctx context.Context, a *app.App, r BuildReq) (presenters.BuildView, error) {
	cfg := r.Config
	if cfg.Builder.Name == "none" {
		return presenters.BuildView{Skipped: true, Reason: "builder is 'none'"}, nil
	}
	if cfg.Builder.Name != "docker" {
		return presenters.BuildView{}, fmt.Errorf("unsupported builder: %s (only 'docker' is currently supported)", cfg.Builder.Name)
	}
	if cfg.Rootfs.Name == "default" {
		return presenters.BuildView{}, fmt.Errorf("If you're trying to upload a custom rootfs, please specify a new name for rootfs.name in vers.toml. Otherwise, set builder.name to 'none'.")
	}
	if _, err := os.Stat(cfg.Builder.Dockerfile); os.IsNotExist(err) {
		return presenters.BuildView{}, fmt.Errorf("Dockerfile '%s' not found in current directory", cfg.Builder.Dockerfile)
	}

	tarBytes, cleanup, err := buildsvc.CreateWorkspaceTar()
	if err != nil {
		return presenters.BuildView{}, err
	}
	defer cleanup()

	name, err := rootfssvc.Upload(ctx, a.Client, cfg.Rootfs.Name, cfg.Builder.Dockerfile, tarBytes)
	if err != nil {
		return presenters.BuildView{}, err
	}

	return presenters.BuildView{RootfsName: name}, nil
}
