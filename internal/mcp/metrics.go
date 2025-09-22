package mcp

import (
	"context"
	"fmt"
	"os"
	"sync"
	"time"

	mcp "github.com/modelcontextprotocol/go-sdk/mcp"
)

type ToolMetrics struct {
	Calls         int64
	Errors        int64
	TotalDuration time.Duration
	LastError     string
}

type rateWindow struct {
	start time.Time
	used  int
}

type rateCfg struct {
	perMinute int
	win       rateWindow
}

var (
	metricsMu     sync.Mutex
	metricsByTool = map[string]*ToolMetrics{}
	rateByTool    = map[string]*rateCfg{}
)

// SetRateLimit sets a simple per-minute limit for a given tool.
func SetRateLimit(tool string, perMinute int) {
	if perMinute <= 0 {
		perMinute = 60
	}
	metricsMu.Lock()
	defer metricsMu.Unlock()
	rateByTool[tool] = &rateCfg{perMinute: perMinute, win: rateWindow{start: time.Now(), used: 0}}
}

func allow(tool string) bool {
	metricsMu.Lock()
	defer metricsMu.Unlock()
	rc, ok := rateByTool[tool]
	if !ok {
		return true
	}
	now := time.Now()
	if now.Sub(rc.win.start) >= time.Minute {
		rc.win.start = now
		rc.win.used = 0
	}
	if rc.win.used >= rc.perMinute {
		return false
	}
	rc.win.used++
	return true
}

func record(tool string, dur time.Duration, err error) {
	metricsMu.Lock()
	defer metricsMu.Unlock()
	tm := metricsByTool[tool]
	if tm == nil {
		tm = &ToolMetrics{}
		metricsByTool[tool] = tm
	}
	tm.Calls++
	tm.TotalDuration += dur
	if err != nil {
		tm.Errors++
		tm.LastError = err.Error()
	}
}

// ToolMetricView is a read-only view for reporting via vers.metrics.
type ToolMetricView struct {
	Calls           int64  `json:"calls"`
	Errors          int64  `json:"errors"`
	TotalDurationMs int64  `json:"totalDurationMs"`
	LastError       string `json:"lastError,omitempty"`
	RatePerMinute   int    `json:"ratePerMinute"`
	UsedInWindow    int    `json:"usedInWindow"`
	WindowResetSec  int64  `json:"windowResetSec"`
}

func snapshotMetrics() map[string]ToolMetricView {
	out := map[string]ToolMetricView{}
	now := time.Now()
	metricsMu.Lock()
	defer metricsMu.Unlock()
	for name, tm := range metricsByTool {
		rv := ToolMetricView{
			Calls:           tm.Calls,
			Errors:          tm.Errors,
			TotalDurationMs: tm.TotalDuration.Milliseconds(),
			LastError:       tm.LastError,
		}
		if rc, ok := rateByTool[name]; ok {
			rv.RatePerMinute = rc.perMinute
			rv.UsedInWindow = rc.win.used
			elapsed := now.Sub(rc.win.start)
			if elapsed >= time.Minute {
				rv.WindowResetSec = 0
			} else {
				rv.WindowResetSec = int64((time.Minute - elapsed).Seconds())
			}
		}
		out[name] = rv
	}
	return out
}

// withMetrics wraps a typed tool handler with rate limiting and metric recording.
func withMetrics[In, Out any](tool string, h func(context.Context, *mcp.CallToolRequest, In) (*mcp.CallToolResult, Out, error)) func(context.Context, *mcp.CallToolRequest, In) (*mcp.CallToolResult, Out, error) {
	return func(ctx context.Context, req *mcp.CallToolRequest, in In) (*mcp.CallToolResult, Out, error) {
		start := time.Now()
		if !allow(tool) {
			err := Err(E_CONFLICT, "rate limit exceeded", map[string]any{"tool": tool})
			fmt.Fprintf(os.Stderr, "[mcp] tool=%s ratelimit exceeded\n", tool)
			var zero Out
			record(tool, time.Since(start), err)
			return nil, zero, err
		}
		res, out, err := h(ctx, req, in)
		record(tool, time.Since(start), err)
		return res, out, err
	}
}
