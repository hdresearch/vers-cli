package handlers

import (
	"context"
	"fmt"
	"os/exec"
	"strings"

	"github.com/hdresearch/vers-cli/internal/app"
	"github.com/hdresearch/vers-cli/internal/auth"
	"github.com/hdresearch/vers-cli/internal/presenters"
	runrt "github.com/hdresearch/vers-cli/internal/runtime"
	vmSvc "github.com/hdresearch/vers-cli/internal/services/vm"
	sshutil "github.com/hdresearch/vers-cli/internal/ssh"
	"github.com/hdresearch/vers-cli/internal/utils"
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

	// Note: State and NetworkInfo no longer available in new SDK
	// Get the host from VERS_URL
	versUrl, err := auth.GetVersUrl()
	if err != nil {
		return v, fmt.Errorf("failed to get host: %w", err)
	}
	sshHost := versUrl.Hostname()
	sshPort := "22"

	cmdStr := strings.Join(r.Command, " ")
	args := sshutil.SSHArgs(sshHost, sshPort, info.KeyPath, cmdStr)

	err = a.Runner.Run(ctx, "ssh", args, runrt.Stdio{Out: a.IO.Out, Err: a.IO.Err})
	if err != nil {
		if ee, ok := err.(*exec.ExitError); ok {
			return v, fmt.Errorf("command exited with code %d", ee.ExitCode())
		}
		return v, fmt.Errorf("failed to run SSH command: %w", err)
	}
	return v, nil
}
