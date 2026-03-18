package mcp

import (
	"context"

	"github.com/hdresearch/vers-cli/internal/app"
	"github.com/hdresearch/vers-cli/internal/handlers"
)

func StatusAdapter(ctx context.Context, a *app.App, in StatusInput) (any, error) {
	return handlers.HandleStatus(ctx, a, handlers.StatusReq{Target: in.Target})
}

func RunAdapter(ctx context.Context, a *app.App, in RunInput) (any, error) {
	req := handlers.RunReq{
		MemSizeMib:  in.MemSizeMib,
		VcpuCount:   in.VcpuCount,
		RootfsName:  in.RootfsName,
		KernelName:  in.KernelName,
		FsSizeVmMib: in.FsSizeVmMib,
		VMAlias:     in.VMAlias,
	}
	return handlers.HandleRun(ctx, a, req)
}

func ExecuteAdapter(ctx context.Context, a *app.App, in ExecuteInput) (any, error) {
	return handlers.HandleExecute(ctx, a, handlers.ExecuteReq{Target: in.Target, Command: in.Command})
}

func BranchAdapter(ctx context.Context, a *app.App, in BranchInput) (any, error) {
	req := handlers.BranchReq{Target: in.Target, Alias: in.Alias, Checkout: in.Checkout, Count: in.Count}
	return handlers.HandleBranch(ctx, a, req)
}

func KillAdapter(ctx context.Context, a *app.App, in KillInput) (any, error) {
	req := handlers.KillReq{
		Targets:          in.Targets,
		SkipConfirmation: in.SkipConfirmation,
	}
	return handlers.HandleKillDTO(ctx, a, req)
}
