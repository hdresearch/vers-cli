package mcp

import (
	"context"
	"fmt"

	"github.com/hdresearch/vers-cli/internal/app"
	mcp "github.com/modelcontextprotocol/go-sdk/mcp"
)

// StartServer wires the MCP server using the Go SDK when built with `-tags mcp`.
// Note: This file references the MCP Go SDK. Ensure the module is added:
//
//	go get github.com/modelcontextprotocol/go-sdk
//
// API calls here are structured to match the SDK’s server + stdio transport.
func StartServer(ctx context.Context, application *app.App, opts Options) error {
	switch opts.Transport {
	case TransportStdio:
		return startStdio(ctx, application, opts)
	case TransportHTTP:
		// Build server and attach tools/resources similarly to stdio path
		server := mcp.NewServer(&mcp.Implementation{Name: "vers", Version: "dev"}, nil)
		if err := registerStatusTool(server, application, opts); err != nil {
			return err
		}
		if err := registerRunTool(server, application, opts); err != nil {
			return err
		}
		if err := registerExecuteTool(server, application, opts); err != nil {
			return err
		}
		if err := registerBranchTool(server, application, opts); err != nil {
			return err
		}
		if err := registerKillTool(server, application, opts); err != nil {
			return err
		}
		if err := registerVersionTool(server, application, opts); err != nil {
			return err
		}
		if err := registerCapabilitiesTool(server, application, opts); err != nil {
			return err
		}
		if err := registerResources(server, application); err != nil {
			return err
		}
		if err := registerMetricsTool(server, application, opts); err != nil {
			return err
		}
		return startHTTP(server, opts.Addr)
	default:
		return fmt.Errorf("unknown transport: %s", opts.Transport)
	}
}

// startStdio initializes a stdio transport MCP server and blocks until context cancel.
func startStdio(ctx context.Context, application *app.App, opts Options) error {
	server := mcp.NewServer(&mcp.Implementation{
		Name:    "vers",
		Version: "dev",
	}, nil)

	if err := registerStatusTool(server, application, opts); err != nil {
		return err
	}
	if err := registerRunTool(server, application, opts); err != nil {
		return err
	}
	if err := registerExecuteTool(server, application, opts); err != nil {
		return err
	}
	if err := registerBranchTool(server, application, opts); err != nil {
		return err
	}
	if err := registerKillTool(server, application, opts); err != nil {
		return err
	}
	if err := registerVersionTool(server, application, opts); err != nil {
		return err
	}
	if err := registerCapabilitiesTool(server, application, opts); err != nil {
		return err
	}
	if err := registerResources(server, application); err != nil {
		return err
	}

	transport := &mcp.StdioTransport{}
	// Optional: emit a simple startup line to stdout/stderr for observability.
	fmt.Fprintln(application.IO.Out, "MCP server (stdio) ready")
	return server.Run(ctx, transport)
}
