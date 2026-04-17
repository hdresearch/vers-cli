package handlers

import (
	"context"
	"fmt"
	"path/filepath"

	"github.com/hdresearch/vers-cli/internal/app"
	"github.com/hdresearch/vers-cli/internal/dockerfile"
	"github.com/hdresearch/vers-cli/internal/presenters"
	"github.com/hdresearch/vers-cli/internal/services/builder"
)

type BuildReq struct {
	Dockerfile  string // path to the Dockerfile (absolute or relative to cwd)
	ContextDir  string // path to the build context root
	Tag         string // optional vers tag to create on the final commit
	NoCache     bool
	Keep        bool
	BuildArgs   map[string]string

	// Machine sizing — required iff FROM scratch.
	MemSizeMib  int64
	VcpuCount   int64
	FsSizeVmMib int64
	RootfsName  string
	KernelName  string
}

func HandleBuild(ctx context.Context, a *app.App, r BuildReq) (presenters.BuildView, error) {
	v := presenters.BuildView{}

	dfPath := r.Dockerfile
	if dfPath == "" {
		dfPath = filepath.Join(r.ContextDir, "Dockerfile")
	} else if !filepath.IsAbs(dfPath) {
		dfPath = filepath.Join(r.ContextDir, dfPath)
	}

	instrs, err := dockerfile.ParseFile(dfPath)
	if err != nil {
		return v, fmt.Errorf("parse %s: %w", dfPath, err)
	}

	bc, err := builder.LoadContext(r.ContextDir)
	if err != nil {
		return v, err
	}

	opts := builder.Options{
		Instructions: instrs,
		Context:      bc,
		BuildArgs:    r.BuildArgs,
		MemSizeMib:   r.MemSizeMib,
		VcpuCount:    r.VcpuCount,
		FsSizeVmMib:  r.FsSizeVmMib,
		RootfsName:   r.RootfsName,
		KernelName:   r.KernelName,
		NoCache:      r.NoCache,
		Keep:         r.Keep,
		Tag:          r.Tag,
	}

	res, err := builder.Build(ctx, a, opts)
	if err != nil {
		return v, err
	}

	v.CommitID = res.FinalCommitID
	v.BuilderVmID = res.BuilderVmID
	v.StepCount = res.StepCount
	v.CachedCount = res.CachedCount
	v.Tag = res.Tag
	v.Cmd = res.Cmd
	v.Entrypoint = res.Entrypoint
	v.ExposedPorts = res.ExposedPorts
	v.Labels = res.Labels
	return v, nil
}
