package mcp

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/hdresearch/vers-cli/internal/app"
	"github.com/hdresearch/vers-cli/internal/presenters"
	mcp "github.com/modelcontextprotocol/go-sdk/mcp"
)

func registerRunTool(server *mcp.Server, application *app.App, opts Options) error {
	tool := &mcp.Tool{
		Name:        "vers.run",
		Description: "Start a VM using inputs similar to vers run",
	}
	handler := withMetrics("vers.run", func(ctx context.Context, req *mcp.CallToolRequest, in RunInput) (*mcp.CallToolResult, presenters.RunView, error) {
		started := time.Now()
		if err := validateRun(in); err != nil {
			return nil, presenters.RunView{}, err
		}
		resAny, err := RunAdapter(ctx, application, in)
		if err != nil {
			err = mapMCPError(err)
			fmt.Fprintf(os.Stderr, "[mcp] tool=vers.run error=%v dur=%s\n", err, time.Since(started).Truncate(time.Millisecond))
			return nil, presenters.RunView{}, err
		}
		res := resAny.(presenters.RunView)
		summary := redact(fmt.Sprintf("VM started: vmID=%s head=%s", res.RootVmID, res.HeadTarget))
		fmt.Fprintf(os.Stderr, "[mcp] tool=vers.run ok dur=%s vmID=%s\n", time.Since(started).Truncate(time.Millisecond), res.RootVmID)
		return &mcp.CallToolResult{Content: []mcp.Content{&mcp.TextContent{Text: summary}}}, res, nil
	})
	mcp.AddTool(server, tool, handler)
	trackTool(tool.Name)
	SetRateLimit(tool.Name, 4)
	return nil
}
