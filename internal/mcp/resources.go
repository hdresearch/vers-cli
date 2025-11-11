package mcp

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"strings"

	"github.com/hdresearch/vers-cli/internal/app"
	"github.com/hdresearch/vers-cli/internal/handlers"
	mcp "github.com/modelcontextprotocol/go-sdk/mcp"
)

func registerResources(server *mcp.Server, application *app.App) error {
	// Status snapshot resource
	server.AddResource(&mcp.Resource{
		Name:        "vers.status",
		Title:       "Vers Status",
		Description: "Status snapshot for all VMs and HEAD VM",
		MIMEType:    "application/json",
		URI:         "vers://status",
	}, readVersResource(application))

	return nil
}

// readVersResource handles vers:// URIs such as:
// - vers://status
func readVersResource(application *app.App) mcp.ResourceHandler {
	return func(ctx context.Context, req *mcp.ReadResourceRequest) (*mcp.ReadResourceResult, error) {
		u, err := url.Parse(req.Params.URI)
		if err != nil {
			return nil, fmt.Errorf("bad URI: %w", err)
		}
		if u.Scheme != "vers" {
			return nil, fmt.Errorf("unsupported scheme: %s", u.Scheme)
		}
		// u.Opaque may be set (vers://something). Prefer Path if present.
		p := strings.TrimPrefix(u.Path, "/")
		if p == "" {
			p = u.Opaque
		}
		parts := strings.Split(p, "/")
		if len(parts) == 0 || parts[0] == "" {
			return nil, fmt.Errorf("missing resource path in URI")
		}
		var payload any
		switch parts[0] {
		case "status":
			var target string
			if len(parts) > 1 {
				target = parts[1]
			}
			apiCtx, cancel := context.WithTimeout(ctx, application.Timeouts.APIMedium)
			defer cancel()
			view, err := handlers.HandleStatus(apiCtx, application, handlers.StatusReq{Target: target})
			if err != nil {
				return nil, mapMCPError(err)
			}
			payload = view
		default:
			return nil, Err(E_NOT_FOUND, "unknown vers resource", map[string]any{"path": p})
		}

		data, err := json.Marshal(payload)
		if err != nil {
			return nil, fmt.Errorf("marshal: %w", err)
		}
		return &mcp.ReadResourceResult{Contents: []*mcp.ResourceContents{{
			URI:      req.Params.URI,
			MIMEType: "application/json",
			Blob:     data,
		}}}, nil
	}
}
