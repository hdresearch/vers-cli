//go:build mcp

package mcp

import (
	"context"
	"testing"
	"time"

	mcp "github.com/modelcontextprotocol/go-sdk/mcp"
)

func TestCapabilitiesListsExpectedTools(t *testing.T) {
	server := mcp.NewServer(&mcp.Implementation{Name: "vers-test", Version: "dev"}, nil)
	// Register the same tools we expose in the real server.
	if err := registerStatusTool(server, nil, Options{}); err != nil {
		t.Fatal(err)
	}
	if err := registerRunTool(server, nil, Options{}); err != nil {
		t.Fatal(err)
	}
	if err := registerExecuteTool(server, nil, Options{}); err != nil {
		t.Fatal(err)
	}
	if err := registerBranchTool(server, nil, Options{}); err != nil {
		t.Fatal(err)
	}
	if err := registerKillTool(server, nil, Options{}); err != nil {
		t.Fatal(err)
	}
	if err := registerVersionTool(server, nil, Options{}); err != nil {
		t.Fatal(err)
	}
	if err := registerCapabilitiesTool(server, nil, Options{}); err != nil {
		t.Fatal(err)
	}
	if err := registerMetricsTool(server, nil, Options{}); err != nil {
		t.Fatal(err)
	}

	t1, t2 := mcp.NewInMemoryTransports()
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	go func() { _ = server.Run(ctx, t2) }()

	client := mcp.NewClient(&mcp.Implementation{Name: "vers-client", Version: "dev"}, nil)
	session, err := client.Connect(ctx, t1, nil)
	if err != nil {
		t.Fatalf("connect: %v", err)
	}
	defer session.Close()

	res, err := session.CallTool(ctx, &mcp.CallToolParams{Name: "vers.capabilities"})
	if err != nil {
		t.Fatalf("CallTool: %v", err)
	}
	out, ok := res.StructuredContent.(map[string]any)
	if !ok {
		t.Fatalf("expected structuredContent map, got %T", res.StructuredContent)
	}
	rawTools, ok := out["tools"].([]any)
	if !ok {
		t.Fatalf("expected tools list, got %T", out["tools"])
	}
	has := func(name string) bool {
		for _, v := range rawTools {
			if s, ok := v.(string); ok && s == name {
				return true
			}
		}
		return false
	}
	expected := []string{"vers.status", "vers.run", "vers.execute", "vers.branch", "vers.kill", "vers.version", "vers.capabilities", "vers.metrics"}
	for _, name := range expected {
		if !has(name) {
			t.Fatalf("capabilities missing tool: %s (got %v)", name, rawTools)
		}
	}
}
