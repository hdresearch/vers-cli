package mcp

import (
	"context"

	"github.com/hdresearch/vers-cli/internal/app"
	mcp "github.com/modelcontextprotocol/go-sdk/mcp"
)

func registerCapabilitiesTool(server *mcp.Server, application *app.App, opts Options) error {
	tool := &mcp.Tool{Name: "vers.capabilities", Description: "List available MCP tools and server settings"}
	handler := func(ctx context.Context, req *mcp.CallToolRequest, _ struct{}) (*mcp.CallToolResult, map[string]any, error) {
		out := map[string]any{
			"tools":     registeredTools,
			"transport": "stdio",
			"verbose":   opts.Verbose,
		}
		return nil, out, nil
	}
	mcp.AddTool(server, tool, handler)
	trackTool(tool.Name)
	return nil
}
