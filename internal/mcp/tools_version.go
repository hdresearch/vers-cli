package mcp

import (
	"context"
	"fmt"

	"github.com/hdresearch/vers-cli/internal/app"
	mcp "github.com/modelcontextprotocol/go-sdk/mcp"
)

// registerVersionTool exposes a no-op introspection endpoint that does not touch the backend.
func registerVersionTool(server *mcp.Server, application *app.App, opts Options) error {
	tool := &mcp.Tool{Name: "vers.version", Description: "Return server and environment info (no backend calls)"}
	handler := func(ctx context.Context, req *mcp.CallToolRequest, _ struct{}) (*mcp.CallToolResult, map[string]any, error) {
		var baseURL string
		if application != nil && application.BaseURL != nil {
			baseURL = application.BaseURL.String()
		}
		out := map[string]any{
			"name":      "vers",
			"version":   "dev",
			"transport": "stdio",
			"baseURL":   baseURL,
			"verbose":   opts.Verbose,
		}
		return &mcp.CallToolResult{Content: []mcp.Content{&mcp.TextContent{Text: fmt.Sprintf("vers mcp server (baseURL=%s)", baseURL)}}}, out, nil
	}
	mcp.AddTool(server, tool, handler)
	trackTool(tool.Name)
	return nil
}
