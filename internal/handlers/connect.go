package handlers

import (
	"context"
	"fmt"
	"os/exec"
	"strings"

	"github.com/hdresearch/vers-cli/internal/app"
	"github.com/hdresearch/vers-cli/internal/presenters"
	runrt "github.com/hdresearch/vers-cli/internal/runtime"
	vmSvc "github.com/hdresearch/vers-cli/internal/services/vm"
	sshutil "github.com/hdresearch/vers-cli/internal/ssh"
	"github.com/hdresearch/vers-cli/internal/utils"
)

type ConnectReq struct{ Target string }

func HandleConnect(ctx context.Context, a *app.App, r ConnectReq) (presenters.ConnectView, error) {
	var ident string
	view := presenters.ConnectView{}
	if strings.TrimSpace(r.Target) == "" {
		headID, err := utils.GetCurrentHeadVM()
		if err != nil {
			return view, fmt.Errorf("no VM ID provided and %w", err)
		}
		view.UsedHEAD = true
		view.HeadID = headID
		ident = headID
	} else {
		ident = r.Target
	}

	info, err := vmSvc.GetConnectInfo(ctx, a.Client, ident)
	if err != nil {
		return view, fmt.Errorf("failed to get VM information: %w", err)
	}

	vmInfo := utils.CreateVMInfoFromVM(*info.VM)
	// Use VM ID as host for SSH-over-TLS (will be formatted as {vm-id}.vm.vers.sh)
	sshHost := info.Host
	sshPort := "443" // SSH-over-TLS uses port 443
	view.LocalRoute = true

	view.VMName = vmInfo.DisplayName
	view.SSHHost = sshHost
	view.SSHPort = sshPort

	// Render connection info BEFORE running SSH so it displays before connecting
	presenters.RenderConnect(a, view)

	args := sshutil.SSHArgs(sshHost, sshPort, info.KeyPath)
	err = a.Runner.Run(ctx, "ssh", args, runrt.Stdio{In: a.IO.In, Out: a.IO.Out, Err: a.IO.Err})
	if err != nil {
		if _, ok := err.(*exec.ExitError); !ok {
			return view, fmt.Errorf("failed to run SSH command: %w", err)
		}
	}
	return view, nil
}
