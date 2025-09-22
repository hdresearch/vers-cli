//go:build mcp

package mcp

import (
	"context"
	"testing"
	"time"

	mcp "github.com/modelcontextprotocol/go-sdk/mcp"
)

// Test calling a no-op tool using in-memory transport to ensure server sessions work.
func TestVersionToolCall_InMemory(t *testing.T) {
	server := mcp.NewServer(&mcp.Implementation{Name: "vers-test", Version: "dev"}, nil)
	if err := registerVersionTool(server, nil, Options{Verbose: false}); err != nil {
		t.Fatalf("registerVersionTool: %v", err)
	}

	t1, t2 := mcp.NewInMemoryTransports()
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Start server
	go func() { _ = server.Run(ctx, t2) }()

	// Connect client
	client := mcp.NewClient(&mcp.Implementation{Name: "vers-client", Version: "dev"}, nil)
	session, err := client.Connect(ctx, t1, nil)
	if err != nil {
		t.Fatalf("client connect: %v", err)
	}
	defer session.Close()

	// Call tool
	res, err := session.CallTool(ctx, &mcp.CallToolParams{Name: "vers.version"})
	if err != nil {
		t.Fatalf("CallTool: %v", err)
	}
	if len(res.Content) == 0 {
		t.Fatalf("expected some content summary")
	}
}
