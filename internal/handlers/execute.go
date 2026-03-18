package handlers

import (
	"context"
	"fmt"

	"github.com/hdresearch/vers-cli/internal/app"
	"github.com/hdresearch/vers-cli/internal/presenters"
	vmSvc "github.com/hdresearch/vers-cli/internal/services/vm"
	sshutil "github.com/hdresearch/vers-cli/internal/ssh"
	"github.com/hdresearch/vers-cli/internal/utils"
	"golang.org/x/crypto/ssh"
)

type ExecuteReq struct {
	Target  string
	Command []string
}

func HandleExecute(ctx context.Context, a *app.App, r ExecuteReq) (presenters.ExecuteView, error) {
	v := presenters.ExecuteView{}

	t, err := utils.ResolveTarget(r.Target)
	if err != nil {
		return v, err
	}
	v.UsedHEAD = t.UsedHEAD
	v.HeadID = t.HeadID

	info, err := vmSvc.GetConnectInfo(ctx, a.Client, t.Ident)
	if err != nil {
		return v, fmt.Errorf("failed to get VM information: %w", err)
	}

	sshHost := info.Host
	cmdStr := utils.ShellJoin(r.Command)

	client := sshutil.NewClient(sshHost, info.KeyPath, info.VMDomain)
	err = client.Execute(ctx, cmdStr, a.IO.Out, a.IO.Err)
	if err != nil {
		if exitErr, ok := err.(*ssh.ExitError); ok {
			return v, fmt.Errorf("command exited with code %d", exitErr.ExitStatus())
		}
		return v, fmt.Errorf("failed to execute command: %w", err)
	}
	return v, nil
}
