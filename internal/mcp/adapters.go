package mcp

import (
	"context"

	"github.com/hdresearch/vers-cli/internal/app"
	"github.com/hdresearch/vers-cli/internal/handlers"
)

// StatusAdapter calls the existing status handler and returns the structured result.
func StatusAdapter(ctx context.Context, a *app.App, in StatusInput) (any, error) {
	return handlers.HandleStatus(ctx, a, handlers.StatusReq{Cluster: in.Cluster, Target: in.Target})
}

// RunAdapter starts a cluster per inputs and returns a presenters.RunView.
func RunAdapter(ctx context.Context, a *app.App, in RunInput) (any, error) {
	req := handlers.RunReq{
		MemSizeMib:       in.MemSizeMib,
		VcpuCount:        in.VcpuCount,
		RootfsName:       in.RootfsName,
		KernelName:       in.KernelName,
		FsSizeClusterMib: in.FsSizeClusterMib,
		FsSizeVmMib:      in.FsSizeVmMib,
		ClusterAlias:     in.ClusterAlias,
		VMAlias:          in.VMAlias,
	}
	return handlers.HandleRun(ctx, a, req)
}

// ExecuteAdapter runs a command in a VM and returns presenters.ExecuteView.
func ExecuteAdapter(ctx context.Context, a *app.App, in ExecuteInput) (any, error) {
	req := handlers.ExecuteReq{Target: in.Target, Command: in.Command}
	return handlers.HandleExecute(ctx, a, req)
}

// BranchAdapter creates a VM branch and returns presenters.BranchView.
func BranchAdapter(ctx context.Context, a *app.App, in BranchInput) (any, error) {
	req := handlers.BranchReq{Target: in.Target, Alias: in.Alias, Checkout: in.Checkout}
	return handlers.HandleBranch(ctx, a, req)
}

// KillAdapter deletes VMs or clusters per inputs. Returns nil output on success.
func KillAdapter(ctx context.Context, a *app.App, in KillInput) (any, error) {
	req := handlers.KillReq{
		Targets:          in.Targets,
		SkipConfirmation: in.SkipConfirmation,
		Recursive:        in.Recursive,
		IsCluster:        in.IsCluster,
		KillAll:          in.KillAll,
	}
	dto, err := handlers.HandleKillDTO(ctx, a, req)
	if err != nil {
		return nil, err
	}
	return dto, nil
}
