package handlers

import (
	"context"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/hdresearch/vers-cli/internal/app"
	"github.com/hdresearch/vers-cli/internal/presenters"
	vmSvc "github.com/hdresearch/vers-cli/internal/services/vm"
	sshutil "github.com/hdresearch/vers-cli/internal/ssh"
	"github.com/hdresearch/vers-cli/internal/utils"
	"golang.org/x/term"
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

	// Get stdin as io.Reader
	var stdin io.Reader
	if a.IO.In != nil {
		stdin = a.IO.In
	} else {
		stdin = os.Stdin
	}

	// Check if stdin is a terminal
	var fd int
	var oldState *term.State
	if f, ok := stdin.(*os.File); ok {
		fd = int(f.Fd())
		if term.IsTerminal(fd) {
			// Save terminal state before SSH so we can restore it if the connection
			// exits abnormally (network drop, server crash, etc.)
			oldState, _ = term.GetState(fd)

			// Put terminal in raw mode for interactive session
			rawState, err := term.MakeRaw(fd)
			if err == nil {
				defer term.Restore(fd, rawState)
			}
		}
	}

	// Use native SSH client
	client := sshutil.NewClient(sshHost, info.KeyPath)
	err = client.Interactive(ctx, stdin, a.IO.Out, a.IO.Err)

	// Always restore terminal state after SSH exits, regardless of how it exited
	if oldState != nil {
		_ = term.Restore(fd, oldState)
	}

	if err != nil {
		// Context cancellation is not an error for interactive sessions
		if ctx.Err() != nil {
			return view, nil
		}
		return view, fmt.Errorf("SSH session failed: %w", err)
	}
	return view, nil
}
