package handlers_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/hdresearch/vers-cli/internal/handlers"
)

func TestHandleInfo(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch {
		case r.URL.Path == "/api/v1/vm/vm-123/status" && r.Method == http.MethodGet:
			// ResolveVMIdentifier calls Status
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`{"vm_id": "vm-123", "owner_id": "owner-1", "created_at": "2026-03-17T00:00:00Z", "state": "Running"}`))
		case r.URL.Path == "/api/v1/vm/vm-123/metadata" && r.Method == http.MethodGet:
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`{
				"vm_id": "vm-123",
				"owner_id": "owner-1",
				"created_at": "2026-03-17T00:00:00Z",
				"state": "Running",
				"ip": "10.0.0.5",
				"parent_commit_id": "commit-abc",
				"grandparent_vm_id": "vm-000"
			}`))
		default:
			t.Errorf("unexpected request: %s %s", r.Method, r.URL.Path)
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer server.Close()

	a := testApp(server.URL)
	res, err := handlers.HandleInfo(context.Background(), a, handlers.InfoReq{Target: "vm-123"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if res.Metadata.VmID != "vm-123" {
		t.Errorf("expected VM ID vm-123, got %s", res.Metadata.VmID)
	}
	if res.Metadata.IP != "10.0.0.5" {
		t.Errorf("expected IP 10.0.0.5, got %s", res.Metadata.IP)
	}
	if string(res.Metadata.State) != "Running" {
		t.Errorf("expected state Running, got %s", res.Metadata.State)
	}
	if res.Metadata.ParentCommitID != "commit-abc" {
		t.Errorf("expected parent commit commit-abc, got %s", res.Metadata.ParentCommitID)
	}
	if res.Metadata.GrandparentVmID != "vm-000" {
		t.Errorf("expected grandparent VM vm-000, got %s", res.Metadata.GrandparentVmID)
	}
	if res.UsedHEAD {
		t.Error("expected UsedHEAD=false when target is provided")
	}
}

func TestHandleInfo_NotFound(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusNotFound)
		w.Write([]byte(`{"error": "VM not found"}`))
	}))
	defer server.Close()

	a := testApp(server.URL)
	_, err := handlers.HandleInfo(context.Background(), a, handlers.InfoReq{Target: "nonexistent"})
	if err == nil {
		t.Fatal("expected error for non-existent VM")
	}
}
