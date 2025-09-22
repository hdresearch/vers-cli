package mcp

import (
	"context"

	"github.com/hdresearch/vers-cli/internal/app"
	mcp "github.com/modelcontextprotocol/go-sdk/mcp"
)

func registerMetricsTool(server *mcp.Server, application *app.App, opts Options) error {
	tool := &mcp.Tool{Name: "vers.metrics", Description: "Return per-tool counters and rate limits"}
	handler := func(ctx context.Context, req *mcp.CallToolRequest, _ struct{}) (*mcp.CallToolResult, map[string]ToolMetricView, error) {
		snap := snapshotMetrics()
		return nil, snap, nil
	}
	mcp.AddTool(server, tool, handler)
	trackTool(tool.Name)
	return nil
}
