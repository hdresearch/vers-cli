package builder

import (
	"context"
	"fmt"
	"io"

	"github.com/hdresearch/vers-cli/internal/app"
	delsvc "github.com/hdresearch/vers-cli/internal/services/deletion"
	vmSvc "github.com/hdresearch/vers-cli/internal/services/vm"
	sshutil "github.com/hdresearch/vers-cli/internal/ssh"
	"github.com/hdresearch/vers-cli/internal/utils"
	vers "github.com/hdresearch/vers-sdk-go"
)

// realExecutor is the production Executor backed by the Vers SDK + SSH.
type realExecutor struct {
	client *vers.Client
}

// NewRealExecutor returns an Executor that drives a live Vers backend via
// the SDK client held by the App container.
func NewRealExecutor(a *app.App) Executor {
	return &realExecutor{client: a.Client}
}

func (e *realExecutor) NewVM(ctx context.Context, spec VMSpec) (string, error) {
	cfg := vers.NewRootRequestVmConfigParam{
		MemSizeMib: vers.F(spec.MemSizeMib),
		VcpuCount:  vers.F(spec.VcpuCount),
		FsSizeMib:  vers.F(spec.FsSizeVmMib),
	}
	if spec.RootfsName != "" {
		cfg.ImageName = vers.F(spec.RootfsName)
	}
	if spec.KernelName != "" {
		cfg.KernelName = vers.F(spec.KernelName)
	}
	resp, err := e.client.Vm.NewRoot(ctx, vers.VmNewRootParams{
		NewRootRequest: vers.NewRootRequestParam{VmConfig: vers.F(cfg)},
	})
	if err != nil {
		return "", err
	}
	if err := utils.WaitForRunning(ctx, e.client, resp.VmID); err != nil {
		return resp.VmID, err
	}
	return resp.VmID, nil
}

func (e *realExecutor) RestoreFromCommit(ctx context.Context, commitID string) (string, error) {
	resp, err := e.client.Vm.RestoreFromCommit(ctx, vers.VmRestoreFromCommitParams{
		VmFromCommitRequest: vers.VmFromCommitRequestParam{CommitID: vers.F(commitID)},
	})
	if err != nil {
		return "", err
	}
	if err := utils.WaitForRunning(ctx, e.client, resp.VmID); err != nil {
		return resp.VmID, err
	}
	return resp.VmID, nil
}

func (e *realExecutor) ResolveTag(ctx context.Context, name string) (string, bool) {
	tag, err := e.client.CommitTags.Get(ctx, name)
	if err != nil || tag == nil || tag.CommitID == "" {
		return "", false
	}
	return tag.CommitID, true
}

func (e *realExecutor) Run(ctx context.Context, vmID string, cmd []string, env map[string]string, workdir string, stdout, stderr io.Writer) (int, error) {
	body, err := vmSvc.ExecStream(ctx, vmID, vmSvc.ExecRequest{
		Command:    cmd,
		Env:        env,
		WorkingDir: workdir,
	})
	if err != nil {
		return -1, err
	}
	defer body.Close()
	return streamOutput(body, stdout, stderr)
}

func (e *realExecutor) Upload(ctx context.Context, vmID, localAbs, remote string, recursive bool) error {
	info, err := vmSvc.GetConnectInfo(ctx, e.client, vmID)
	if err != nil {
		return fmt.Errorf("connect info: %w", err)
	}
	c := sshutil.NewClient(info.Host, info.KeyPath, info.VMDomain)
	return c.Upload(ctx, localAbs, remote, recursive)
}

func (e *realExecutor) Commit(ctx context.Context, vmID string) (string, error) {
	resp, err := e.client.Vm.Commit(ctx, vmID, vers.VmCommitParams{})
	if err != nil {
		return "", err
	}
	return resp.CommitID, nil
}

func (e *realExecutor) CreateTag(ctx context.Context, name, commitID string) error {
	_, err := e.client.CommitTags.New(ctx, vers.CommitTagNewParams{
		CreateTagRequest: vers.CreateTagRequestParam{
			TagName:  vers.F(name),
			CommitID: vers.F(commitID),
		},
	})
	return err
}

func (e *realExecutor) DeleteVM(ctx context.Context, vmID string) error {
	_, err := delsvc.DeleteVM(ctx, e.client, vmID)
	return err
}
