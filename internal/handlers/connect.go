package handlers

import (
	"context"
	"fmt"
	"io"
	"os"
	"time"

	"github.com/hdresearch/vers-cli/internal/app"
	"github.com/hdresearch/vers-cli/internal/presenters"
	vmSvc "github.com/hdresearch/vers-cli/internal/services/vm"
	sshutil "github.com/hdresearch/vers-cli/internal/ssh"
	"github.com/hdresearch/vers-cli/internal/utils"
	vers "github.com/hdresearch/vers-sdk-go"
	"golang.org/x/crypto/ssh"
	"golang.org/x/term"
)

type ConnectReq struct{ Target string }

func HandleConnect(ctx context.Context, a *app.App, r ConnectReq) (presenters.ConnectView, error) {
	view := presenters.ConnectView{}

	t, err := utils.ResolveTarget(r.Target)
	if err != nil {
		return view, err
	}
	view.UsedHEAD = t.UsedHEAD
	view.HeadID = t.HeadID

	info, err := vmSvc.GetConnectInfo(ctx, a.Client, t.Ident)
	if err != nil {
		return view, fmt.Errorf("failed to get VM information: %w", err)
	}

	// Check VM state before attempting to connect
	switch info.VM.State {
	case vers.VmStateRunning:
		// Good to go
	case vers.VmStatePaused:
		fmt.Fprintln(a.IO.Out, "VM is paused. Resuming...")
		updateParams := vers.VmUpdateStateParams{
			VmUpdateStateRequest: vers.VmUpdateStateRequestParam{
				State: vers.F(vers.VmUpdateStateRequestStateRunning),
			},
		}
		if err := a.Client.Vm.UpdateState(ctx, info.VM.VmID, updateParams); err != nil {
			return view, fmt.Errorf("failed to resume VM: %w", err)
		}
		// Wait briefly for the VM to become ready
		time.Sleep(2 * time.Second)
	case vers.VmStateBooting:
		fmt.Fprintln(a.IO.Out, "VM is still booting. Waiting...")
		// Poll until the VM is running
		for i := 0; i < 30; i++ {
			time.Sleep(2 * time.Second)
			refreshed, _, err := utils.GetVmAndNodeIP(ctx, a.Client, info.VM.VmID)
			if err != nil {
				return view, fmt.Errorf("failed to check VM status: %w", err)
			}
			if refreshed.State == vers.VmStateRunning {
				info.VM = refreshed
				break
			}
			if refreshed.State == vers.VmStatePaused {
				return view, fmt.Errorf("VM entered paused state while booting — try 'vers resume %s' first", t.Ident)
			}
			if i == 29 {
				return view, fmt.Errorf("timed out waiting for VM to finish booting")
			}
		}
	default:
		return view, fmt.Errorf("VM is in '%s' state and cannot be connected to", info.VM.State)
	}

	vmInfo := utils.CreateVMInfoFromVM(*info.VM)
	// Use VM ID as host for SSH-over-TLS (will be formatted as {vm-id}.{vmDomain})
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

	// Use native SSH client with automatic reconnection on dropped connections
	client := sshutil.NewClient(sshHost, info.KeyPath, info.VMDomain)

	const maxReconnectAttempts = 5
	const initialBackoff = 1 * time.Second
	const maxBackoff = 10 * time.Second

	for attempt := 0; ; attempt++ {
		err = client.Interactive(ctx, stdin, a.IO.Out, a.IO.Err)

		// Always restore terminal state after SSH exits
		if oldState != nil {
			_ = term.Restore(fd, oldState)
		}

		// Clean exit (user typed exit/logout) — done
		if err == nil {
			return view, nil
		}

		// Context cancelled (e.g. vers process killed) — done
		if ctx.Err() != nil {
			return view, nil
		}

		// User's shell exited with a status code — done (not a dropped connection)
		if _, ok := err.(*ssh.ExitError); ok {
			return view, nil
		}

		// Connection dropped — attempt reconnect
		if attempt >= maxReconnectAttempts {
			return view, fmt.Errorf("SSH connection lost after %d reconnection attempts: %w", maxReconnectAttempts, err)
		}

		backoff := initialBackoff * time.Duration(1<<uint(attempt))
		if backoff > maxBackoff {
			backoff = maxBackoff
		}

		fmt.Fprintf(a.IO.Out, "\r\nConnection lost. Reconnecting (attempt %d/%d)...\r\n", attempt+1, maxReconnectAttempts)
		if f, ok := a.IO.Out.(*os.File); ok {
			f.Sync()
		}

		select {
		case <-ctx.Done():
			return view, nil
		case <-time.After(backoff):
		}

		// Re-enter raw mode for next interactive session
		if oldState != nil {
			if _, rawErr := term.MakeRaw(fd); rawErr != nil {
				return view, fmt.Errorf("failed to restore raw terminal for reconnect: %w", rawErr)
			}
		}
	}
}
