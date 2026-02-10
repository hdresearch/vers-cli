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
)

// TunnelReq holds the parameters for a tunnel command.
type TunnelReq struct {
	Target     string // VM ID, alias, or empty for HEAD
	LocalPort  int    // local port to listen on (0 = auto)
	RemoteHost string // remote host (from the VM's perspective), default "localhost"
	RemotePort int    // remote port to forward to
}

// HandleTunnel sets up an SSH port-forwarding tunnel to the VM.
// It blocks until ctx is cancelled (e.g. Ctrl-C).
func HandleTunnel(ctx context.Context, a *app.App, r TunnelReq) (presenters.TunnelView, error) {
	view := presenters.TunnelView{}

	// Resolve VM target
	var ident string
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

	// Default remote host
	if r.RemoteHost == "" {
		r.RemoteHost = "localhost"
	}

	// Get connection info (resolves alias, checks VM exists, gets SSH key)
	info, err := vmSvc.GetConnectInfo(ctx, a.Client, ident)
	if err != nil {
		return view, fmt.Errorf("failed to get VM information: %w", err)
	}

	// Set up the tunnel
	client := sshutil.NewClient(info.Host, info.KeyPath)
	tunnel, err := client.Forward(ctx, r.LocalPort, r.RemoteHost, r.RemotePort)
	if err != nil {
		return view, fmt.Errorf("failed to start tunnel: %w", err)
	}

	view.VMName = info.VM.VmID
	view.LocalPort = tunnel.LocalPort
	view.RemoteHost = r.RemoteHost
	view.RemotePort = r.RemotePort

	// Render before blocking
	presenters.RenderTunnel(a, view)

	// Block until context is cancelled (Ctrl-C)
	<-ctx.Done()
	tunnel.Close()

	return view, nil
}
