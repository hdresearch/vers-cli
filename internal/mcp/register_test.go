package mcp

import (
	"testing"

	mcp "github.com/modelcontextprotocol/go-sdk/mcp"
)

// This is a smoke test that the tools register successfully and schemas resolve.
func TestRegisterTools(t *testing.T) {
	server := mcp.NewServer(&mcp.Implementation{Name: "vers-test", Version: "dev"}, nil)
	var appNil = (*struct{})(nil) // application isn't used until handler call
	_ = appNil

	if err := registerStatusTool(server, nil, Options{}); err != nil {
		t.Fatalf("registerStatusTool: %v", err)
	}
	if err := registerRunTool(server, nil, Options{}); err != nil {
		t.Fatalf("registerRunTool: %v", err)
	}
	if err := registerExecuteTool(server, nil, Options{}); err != nil {
		t.Fatalf("registerExecuteTool: %v", err)
	}
	if err := registerBranchTool(server, nil, Options{}); err != nil {
		t.Fatalf("registerBranchTool: %v", err)
	}
	if err := registerKillTool(server, nil, Options{}); err != nil {
		t.Fatalf("registerKillTool: %v", err)
	}
}
