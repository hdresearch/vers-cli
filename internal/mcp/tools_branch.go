package mcp

import (
	"context"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/hdresearch/vers-cli/internal/app"
	"github.com/hdresearch/vers-cli/internal/presenters"
	mcp "github.com/modelcontextprotocol/go-sdk/mcp"
)

func registerBranchTool(server *mcp.Server, application *app.App, opts Options) error {
	tool := &mcp.Tool{
		Name:        "vers.branch",
		Description: "Create a new VM (branch) from an existing VM or HEAD",
	}
	handler := withMetrics("vers.branch", func(ctx context.Context, req *mcp.CallToolRequest, in BranchInput) (*mcp.CallToolResult, presenters.BranchView, error) {
		started := time.Now()
		if err := validateBranch(in); err != nil {
			return nil, presenters.BranchView{}, err
		}
		resAny, err := BranchAdapter(ctx, application, in)
		if err != nil {
			err = mapMCPError(err)
			fmt.Fprintf(os.Stderr, "[mcp] tool=vers.branch error=%v dur=%s\n", err, time.Since(started).Truncate(time.Millisecond))
			return nil, presenters.BranchView{}, err
		}
		res := resAny.(presenters.BranchView)
		display := res.NewAlias
		if display == "" {
			switch {
			case len(res.NewIDs) > 0:
				display = strings.Join(res.NewIDs, ", ")
			default:
				display = res.NewID
			}
		}
		summary := redact(fmt.Sprintf("branch created: new=%s from=%s checkout=%t", display, res.FromID, res.CheckoutDone))
		fmt.Fprintf(os.Stderr, "[mcp] tool=vers.branch ok dur=%s new=%s\n", time.Since(started).Truncate(time.Millisecond), display)
		return &mcp.CallToolResult{Content: []mcp.Content{&mcp.TextContent{Text: summary}}}, res, nil
	})
	mcp.AddTool(server, tool, handler)
	trackTool(tool.Name)
	SetRateLimit(tool.Name, 10)
	return nil
}
