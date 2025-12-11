package handlers

import (
	"context"
	"fmt"
	"strings"

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
	var ident string
	if r.Target == "" {
		head, err := utils.GetCurrentHeadVM()
		if err != nil {
			return v, fmt.Errorf("no VM ID provided and %w", err)
		}
		v.UsedHEAD = true
		v.HeadID = head
		ident = head
	} else {
		ident = r.Target
	}

	info, err := vmSvc.GetConnectInfo(ctx, a.Client, ident)
	if err != nil {
		return v, fmt.Errorf("failed to get VM information: %w", err)
	}

	// Use VM ID as host for SSH-over-TLS (will be formatted as {vm-id}.vm.vers.sh)
	sshHost := info.Host

	cmdStr := strings.Join(r.Command, " ")

	// Use native SSH client
	client := sshutil.NewClient(sshHost, info.KeyPath)
	err = client.Execute(ctx, cmdStr, a.IO.Out, a.IO.Err)
	if err != nil {
		// Check for SSH exit error to get exit code
		if exitErr, ok := err.(*ssh.ExitError); ok {
			return v, fmt.Errorf("command exited with code %d", exitErr.ExitStatus())
		}
		return v, fmt.Errorf("failed to execute command: %w", err)
	}
	return v, nil
}
