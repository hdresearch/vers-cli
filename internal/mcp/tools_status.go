package mcp

import (
	"context"
	"fmt"

	"github.com/hdresearch/vers-cli/internal/app"
	"github.com/hdresearch/vers-cli/internal/handlers"
	"github.com/hdresearch/vers-cli/internal/presenters"
	mcp "github.com/modelcontextprotocol/go-sdk/mcp"
)

// registerStatusTool registers the vers.status tool with the MCP server.
// Expected to be called from StartServer after server construction.
//
// Pseudocode (requires actual SDK types):
//
//	server.AddTool(mcpserver.Tool{
//	    Name:        "vers.status",
//	    Description: "Get VM status",
//	    InputSchema:  <json-schema>,
//	}, handler)
//
// The handler parses input into StatusInput and calls StatusAdapter.
func registerStatusTool(server *mcp.Server, application *app.App, opts Options) error {
	tool := &mcp.Tool{
		Name:        "vers.status",
		Description: "Get VM status. Optional: target (VM ID or alias)",
	}

	handler := withMetrics("vers.status", func(ctx context.Context, req *mcp.CallToolRequest, in StatusInput) (*mcp.CallToolResult, presenters.StatusView, error) {
		// Propagate reasonable timeout; reuse existing app timeouts.
		apiCtx, cancel := context.WithTimeout(ctx, application.Timeouts.APIMedium)
		defer cancel()

		res, err := handlers.HandleStatus(apiCtx, application, handlers.StatusReq{Target: in.Target})
		if err != nil {
			return nil, presenters.StatusView{}, mapMCPError(err)
		}

		// Provide a short, human-friendly text summary alongside structured JSON output.
		summary := fmt.Sprintf("status: mode=%d vms=%d", res.Mode, len(res.VMs))
		return &mcp.CallToolResult{
			Content: []mcp.Content{&mcp.TextContent{Text: summary}},
		}, res, nil
	})

	mcp.AddTool(server, tool, handler)
	trackTool(tool.Name)
	SetRateLimit(tool.Name, 60)
	return nil
}

// handleStatus is a convenience wrapper showing the adapter usage.
func handleStatus(ctx context.Context, application *app.App, in StatusInput) (string, any, error) {
	res, err := StatusAdapter(ctx, application, in)
	if err != nil {
		return "", nil, err
	}
	return "Status fetched", res, nil
}
