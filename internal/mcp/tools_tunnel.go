package mcp

import (
	"context"
	"fmt"
	"os"
	"sync"
	"time"

	"github.com/hdresearch/vers-cli/internal/app"
	"github.com/hdresearch/vers-cli/internal/presenters"
	vmSvc "github.com/hdresearch/vers-cli/internal/services/vm"
	sshutil "github.com/hdresearch/vers-cli/internal/ssh"
	"github.com/hdresearch/vers-cli/internal/utils"
	mcp "github.com/modelcontextprotocol/go-sdk/mcp"
)

func registerTunnelTool(server *mcp.Server, application *app.App, opts Options) error {
	tool := &mcp.Tool{
		Name:        "vers.tunnel",
		Description: "Open an SSH tunnel forwarding a local port to a port on a VM (HEAD if target omitted). Returns the local port being listened on. The tunnel stays open until vers.tunnel.close is called or the MCP session ends.",
	}
	handler := withMetrics("vers.tunnel", func(ctx context.Context, req *mcp.CallToolRequest, in TunnelInput) (*mcp.CallToolResult, presenters.TunnelView, error) {
		if err := validateTunnel(in); err != nil {
			return nil, presenters.TunnelView{}, err
		}
		start := time.Now()

		// Resolve target (falls back to HEAD if empty)
		v := presenters.TunnelView{}
		resolved, err := utils.ResolveTarget(in.Target)
		if err != nil {
			return nil, v, err
		}
		v.UsedHEAD = resolved.UsedHEAD
		v.HeadID = resolved.HeadID

		info, err := vmSvc.GetConnectInfo(ctx, application.Client, resolved.Ident)
		if err != nil {
			return nil, v, mapMCPError(fmt.Errorf("failed to get VM information: %w", err))
		}

		remoteHost := in.RemoteHost
		if remoteHost == "" {
			remoteHost = "localhost"
		}

		// Open the tunnel
		client := sshutil.NewClient(info.Host, info.KeyPath)
		tunnel, err := client.Forward(ctx, in.LocalPort, remoteHost, in.RemotePort)
		if err != nil {
			return nil, v, fmt.Errorf("failed to start tunnel: %w", err)
		}

		duration := time.Since(start)
		target := resolved.Ident

		v.VMName = info.VM.VmID
		v.LocalPort = tunnel.LocalPort
		v.RemoteHost = remoteHost
		v.RemotePort = in.RemotePort

		// Store the tunnel so it can be closed later
		storeTunnel(tunnel)

		summary := fmt.Sprintf("Tunnel open: 127.0.0.1:%d → %s:%d on VM %s (established in %s)",
			tunnel.LocalPort, remoteHost, in.RemotePort, target, duration.Truncate(time.Millisecond))
		fmt.Fprintf(os.Stderr, "[mcp] tool=vers.tunnel ok dur=%s target=%s local=%d remote=%s:%d\n",
			duration.Truncate(time.Millisecond), target, tunnel.LocalPort, remoteHost, in.RemotePort)

		return &mcp.CallToolResult{Content: []mcp.Content{&mcp.TextContent{Text: summary}}}, v, nil
	})
	mcp.AddTool(server, tool, handler)
	trackTool(tool.Name)
	SetRateLimit(tool.Name, 10)

	// Also register a close tool
	closeTool := &mcp.Tool{
		Name:        "vers.tunnel.close",
		Description: "Close all open SSH tunnels.",
	}
	closeHandler := withMetrics("vers.tunnel.close", func(ctx context.Context, req *mcp.CallToolRequest, in struct{}) (*mcp.CallToolResult, struct{}, error) {
		n := closeAllTunnels()
		summary := fmt.Sprintf("Closed %d tunnel(s)", n)
		fmt.Fprintf(os.Stderr, "[mcp] tool=vers.tunnel.close closed=%d\n", n)
		return &mcp.CallToolResult{Content: []mcp.Content{&mcp.TextContent{Text: summary}}}, struct{}{}, nil
	})
	mcp.AddTool(server, closeTool, closeHandler)
	trackTool(closeTool.Name)

	return nil
}

// --- tunnel registry ---

var (
	activeTunnels   []*sshutil.Tunnel
	activeTunnelsMu = &sync.Mutex{}
)

func storeTunnel(t *sshutil.Tunnel) {
	activeTunnelsMu.Lock()
	defer activeTunnelsMu.Unlock()
	activeTunnels = append(activeTunnels, t)
}

func closeAllTunnels() int {
	activeTunnelsMu.Lock()
	defer activeTunnelsMu.Unlock()
	n := len(activeTunnels)
	for _, t := range activeTunnels {
		t.Close()
	}
	activeTunnels = nil
	return n
}
