package ssh

import (
	"context"
	"testing"
	"time"
)

// TestClient_Connect_InvalidHost tests that connection to an invalid host fails gracefully.
func TestClient_Connect_InvalidHost(t *testing.T) {
	client := NewClient("nonexistent-vm-12345", "/nonexistent/key")

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	_, err := client.Connect(ctx)
	if err == nil {
		t.Fatal("expected error connecting to invalid host, got nil")
	}
	// Should fail either on key read or TLS dial
	t.Logf("got expected error: %v", err)
}

// TestClient_Execute_InvalidHost tests that execute to an invalid host fails gracefully.
func TestClient_Execute_InvalidHost(t *testing.T) {
	client := NewClient("nonexistent-vm-12345", "/nonexistent/key")

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	err := client.Execute(ctx, "echo hello", nil, nil)
	if err == nil {
		t.Fatal("expected error executing on invalid host, got nil")
	}
	t.Logf("got expected error: %v", err)
}
