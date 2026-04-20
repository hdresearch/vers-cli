package handlers_test

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/hdresearch/vers-cli/internal/handlers"
)

// TestHandleRunCommit_CommitIDPath verifies that when IsRef is false, the
// request body places the argument under "commit_id" (no "ref" key).
// This is the default behavior — any existing callers must not regress.
func TestHandleRunCommit_CommitIDPath(t *testing.T) {
	var receivedBody map[string]interface{}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v1/vm/from_commit" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		if r.Method != http.MethodPost {
			t.Errorf("expected POST, got %s", r.Method)
		}
		body, _ := io.ReadAll(r.Body)
		json.Unmarshal(body, &receivedBody)

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"vm_id": "vm-uuid-1"}`))
	}))
	defer server.Close()

	a := testApp(server.URL)
	res, err := handlers.HandleRunCommit(context.Background(), a, handlers.RunCommitReq{
		CommitKey: "abc-123",
		IsRef:     false,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if res.RootVmID != "vm-uuid-1" {
		t.Errorf("expected VM ID vm-uuid-1, got %s", res.RootVmID)
	}
	if res.CommitKey != "abc-123" {
		t.Errorf("expected CommitKey abc-123, got %s", res.CommitKey)
	}

	// The SDK flattens the vm_from_commit_request union into the top-level
	// request body (MarshalJSON on VmRestoreFromCommitParams inlines it).
	if receivedBody["commit_id"] != "abc-123" {
		t.Errorf("expected commit_id=abc-123, got %v", receivedBody["commit_id"])
	}
	if _, refPresent := receivedBody["ref"]; refPresent {
		t.Errorf("expected no 'ref' key when IsRef=false, got: %v", receivedBody)
	}
}

// TestHandleRunCommit_RefPath verifies that when IsRef is true, the request
// body uses "ref" instead of "commit_id". This is the new behavior added
// by the --ref flag.
func TestHandleRunCommit_RefPath(t *testing.T) {
	var receivedBody map[string]interface{}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v1/vm/from_commit" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		if r.Method != http.MethodPost {
			t.Errorf("expected POST, got %s", r.Method)
		}
		body, _ := io.ReadAll(r.Body)
		json.Unmarshal(body, &receivedBody)

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"vm_id": "vm-uuid-2"}`))
	}))
	defer server.Close()

	a := testApp(server.URL)
	res, err := handlers.HandleRunCommit(context.Background(), a, handlers.RunCommitReq{
		CommitKey: "my-app:latest",
		IsRef:     true,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if res.RootVmID != "vm-uuid-2" {
		t.Errorf("expected VM ID vm-uuid-2, got %s", res.RootVmID)
	}
	if res.CommitKey != "my-app:latest" {
		t.Errorf("expected CommitKey my-app:latest, got %s", res.CommitKey)
	}

	// The SDK flattens the vm_from_commit_request union into the top-level
	// request body (MarshalJSON on VmRestoreFromCommitParams inlines it).
	if receivedBody["ref"] != "my-app:latest" {
		t.Errorf("expected ref=my-app:latest, got %v", receivedBody["ref"])
	}
	if _, commitIDPresent := receivedBody["commit_id"]; commitIDPresent {
		t.Errorf("expected no 'commit_id' key when IsRef=true, got: %v", receivedBody)
	}
}

// TestHandleRunCommit_ServerError verifies that non-2xx responses surface as
// errors to the caller (no silent swallowing).
func TestHandleRunCommit_ServerError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusUnprocessableEntity)
		w.Write([]byte(`{"error": "invalid commit id"}`))
	}))
	defer server.Close()

	a := testApp(server.URL)
	_, err := handlers.HandleRunCommit(context.Background(), a, handlers.RunCommitReq{
		CommitKey: "not-a-uuid",
		IsRef:     false,
	})
	if err == nil {
		t.Fatal("expected error for 422 response, got nil")
	}
}
