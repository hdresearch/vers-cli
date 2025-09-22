package mcp

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/hdresearch/vers-cli/internal/app"
	"github.com/hdresearch/vers-cli/internal/handlers"
	mcp "github.com/modelcontextprotocol/go-sdk/mcp"
)

func registerKillTool(server *mcp.Server, application *app.App, opts Options) error {
	tool := &mcp.Tool{
		Name:        "vers.kill",
		Description: "Delete VMs or clusters; when no targets, deletes HEAD VM",
	}
	handler := withMetrics("vers.kill", func(ctx context.Context, req *mcp.CallToolRequest, in KillInput) (*mcp.CallToolResult, handlers.KillDTO, error) {
		started := time.Now()
		if err := validateKill(in); err != nil {
			return nil, handlers.KillDTO{}, err
		}
		dtoAny, err := KillAdapter(ctx, application, in)
		if err != nil {
			err = mapMCPError(err)
			fmt.Fprintf(os.Stderr, "[mcp] tool=vers.kill error=%v dur=%s\n", err, time.Since(started).Truncate(time.Millisecond))
			return nil, handlers.KillDTO{}, err
		}
		dto := dtoAny.(handlers.KillDTO)
		var scope string
		switch {
		case in.KillAll:
			scope = "all-clusters"
		case in.IsCluster:
			scope = "clusters"
		default:
			scope = "vms"
		}
		summary := redact(fmt.Sprintf("deleted %s targets=%v recursive=%t", scope, in.Targets, in.Recursive))
		fmt.Fprintf(os.Stderr, "[mcp] tool=vers.kill ok dur=%s scope=%s count=%d\n", time.Since(started).Truncate(time.Millisecond), scope, len(in.Targets))
		return &mcp.CallToolResult{Content: []mcp.Content{&mcp.TextContent{Text: summary}}}, dto, nil
	})
	mcp.AddTool(server, tool, handler)
	trackTool(tool.Name)
	SetRateLimit(tool.Name, 4)
	return nil
}
