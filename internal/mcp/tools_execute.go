package mcp

import (
	"context"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/hdresearch/vers-cli/internal/app"
	"github.com/hdresearch/vers-cli/internal/presenters"
	runrt "github.com/hdresearch/vers-cli/internal/runtime"
	vmSvc "github.com/hdresearch/vers-cli/internal/services/vm"
	sshutil "github.com/hdresearch/vers-cli/internal/ssh"
	"github.com/hdresearch/vers-cli/internal/utils"
	mcp "github.com/modelcontextprotocol/go-sdk/mcp"
)

func registerExecuteTool(server *mcp.Server, application *app.App, opts Options) error {
	tool := &mcp.Tool{
		Name:        "vers.execute",
		Description: "Execute a command in a VM (HEAD if target omitted)",
	}
	handler := withMetrics("vers.execute", func(ctx context.Context, req *mcp.CallToolRequest, in ExecuteInput) (*mcp.CallToolResult, presenters.ExecuteView, error) {
		if err := validateExecute(in); err != nil {
			return nil, presenters.ExecuteView{}, err
		}
		start := time.Now()

		// Resolve target similar to handlers.HandleExecute but without writing to stdout.
		v := presenters.ExecuteView{}
		ident := in.Target
		if ident == "" {
			head, err := utils.GetCurrentHeadVM()
			if err != nil {
				return nil, presenters.ExecuteView{}, fmt.Errorf("no VM ID provided and %w", err)
			}
			v.UsedHEAD = true
			v.HeadID = head
			ident = head
		}

		info, err := vmSvc.GetConnectInfo(ctx, application.Client, ident)
		if err != nil {
			return nil, presenters.ExecuteView{}, mapMCPError(fmt.Errorf("failed to get VM information: %w", err))
		}
		// Note: State and NetworkInfo no longer available in new SDK
		// Using VM IP and default SSH port
		sshHost := info.VM.IP
		sshPort := "22"

		// Apply timeout override if provided.
		runCtx := ctx
		var cancel func()
		if in.TimeoutSeconds > 0 {
			runCtx, cancel = context.WithTimeout(ctx, time.Duration(in.TimeoutSeconds)*time.Second)
		} else {
			// Default to APIMedium ceiling if available.
			runCtx, cancel = context.WithTimeout(ctx, application.Timeouts.APIMedium)
		}
		defer cancel()

		// Stream stdout/stderr via session logs.
		outw := &sessionWriter{session: req.Session, level: "info"}
		errw := &sessionWriter{session: req.Session, level: "error"}

		args := sshutil.SSHArgs(sshHost, sshPort, info.KeyPath, joinCmd(in.Command))
		runErr := application.Runner.Run(runCtx, "ssh", args, runrt.Stdio{Out: outw, Err: errw})
		duration := time.Since(start)
		target := ident
		if v.UsedHEAD && in.Target == "" {
			target = v.HeadID
		}

		if runErr != nil {
			fmt.Fprintf(os.Stderr, "[mcp] tool=vers.execute error=%v dur=%s target=%s\n", runErr, duration.Truncate(time.Millisecond), target)
			// Return error; logs already streamed.
			return nil, v, fmt.Errorf("execute failed on %s after %s: %v", target, duration.Truncate(time.Millisecond), runErr)
		}

		summary := redact(fmt.Sprintf("execute completed on target=%s in %s", target, duration.Truncate(time.Millisecond)))
		fmt.Fprintf(os.Stderr, "[mcp] tool=vers.execute ok dur=%s target=%s\n", duration.Truncate(time.Millisecond), target)
		return &mcp.CallToolResult{Content: []mcp.Content{&mcp.TextContent{Text: summary}}}, v, nil
	})
	mcp.AddTool(server, tool, handler)
	trackTool(tool.Name)
	SetRateLimit(tool.Name, 10)
	return nil
}

// sessionWriter sends each write as a log message to the MCP session.
type sessionWriter struct {
	session *mcp.ServerSession
	level   string
}

func (w *sessionWriter) Write(p []byte) (int, error) {
	// Best-effort: send as-is; client can render as stream.
	// Avoid tight loops on very large writes by chunking lines roughly.
	msg := redact(string(p))
	_ = w.session.Log(context.Background(), &mcp.LoggingMessageParams{Data: msg, Level: mcp.LoggingLevel(w.level)})
	return len(p), nil
}

func joinCmd(parts []string) string {
	if len(parts) == 0 {
		return ""
	}
	// Minimal shell escaping: join with spaces; SSH will run via sh -lc on remote depending on server side.
	// For safety, users should pass a full command array.
	return strings.Join(parts, " ")
}
