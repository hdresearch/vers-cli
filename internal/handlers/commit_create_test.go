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

func TestHandleCommitCreate_WithName(t *testing.T) {
	var patchBody map[string]interface{}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		switch {
		case r.Method == http.MethodGet && r.URL.Path == "/api/v1/vm/vm-123/status":
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`{"vm_id":"vm-123","owner_id":"owner-1","created_at":"2026-01-01T00:00:00Z","state":"running"}`))

		case r.Method == http.MethodPost && r.URL.Path == "/api/v1/vm/vm-123/commit":
			w.WriteHeader(http.StatusCreated)
			w.Write([]byte(`{"commit_id":"commit-abc"}`))

		case r.Method == http.MethodPatch && r.URL.Path == "/api/v1/commits/commit-abc":
			body, _ := io.ReadAll(r.Body)
			json.Unmarshal(body, &patchBody)
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`{
				"commit_id": "commit-abc",
				"name": "my-commit",
				"owner_id": "owner-1",
				"created_at": "2026-01-01T00:00:00Z",
				"is_public": false,
				"description": "my description"
			}`))

		default:
			t.Errorf("unexpected request: %s %s", r.Method, r.URL.Path)
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer server.Close()

	a := testApp(server.URL)
	res, err := handlers.HandleCommitCreate(context.Background(), a, handlers.CommitCreateReq{
		Target:      "vm-123",
		Name:        "my-commit",
		Description: "my description",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if res.CommitID != "commit-abc" {
		t.Errorf("expected commit ID commit-abc, got %s", res.CommitID)
	}
	if res.VmID != "vm-123" {
		t.Errorf("expected VM ID vm-123, got %s", res.VmID)
	}
	if res.Name != "my-commit" {
		t.Errorf("expected name my-commit, got %s", res.Name)
	}
	if res.Description != "my description" {
		t.Errorf("expected description 'my description', got %s", res.Description)
	}

	// Verify the PATCH request included name and description
	if patchBody == nil {
		t.Fatal("expected PATCH request to be made")
	}
	if patchBody["name"] != "my-commit" {
		t.Errorf("expected PATCH body name=my-commit, got %v", patchBody["name"])
	}
	if patchBody["description"] != "my description" {
		t.Errorf("expected PATCH body description='my description', got %v", patchBody["description"])
	}
}

func TestHandleCommitCreate_WithoutName(t *testing.T) {
	patchCalled := false

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		switch {
		case r.Method == http.MethodGet && r.URL.Path == "/api/v1/vm/vm-123/status":
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`{"vm_id":"vm-123","owner_id":"owner-1","created_at":"2026-01-01T00:00:00Z","state":"running"}`))

		case r.Method == http.MethodPost && r.URL.Path == "/api/v1/vm/vm-123/commit":
			w.WriteHeader(http.StatusCreated)
			w.Write([]byte(`{"commit_id":"commit-abc"}`))

		case r.Method == http.MethodPatch:
			patchCalled = true
			t.Error("PATCH should not be called when no name/description provided")

		default:
			t.Errorf("unexpected request: %s %s", r.Method, r.URL.Path)
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer server.Close()

	a := testApp(server.URL)
	res, err := handlers.HandleCommitCreate(context.Background(), a, handlers.CommitCreateReq{
		Target: "vm-123",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if res.CommitID != "commit-abc" {
		t.Errorf("expected commit ID commit-abc, got %s", res.CommitID)
	}
	if patchCalled {
		t.Error("PATCH should not have been called")
	}
	if res.Name != "" {
		t.Errorf("expected empty name, got %s", res.Name)
	}
}

func TestHandleCommitCreate_NameOnly(t *testing.T) {
	var patchBody map[string]interface{}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		switch {
		case r.Method == http.MethodGet && r.URL.Path == "/api/v1/vm/vm-123/status":
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`{"vm_id":"vm-123","owner_id":"owner-1","created_at":"2026-01-01T00:00:00Z","state":"running"}`))

		case r.Method == http.MethodPost && r.URL.Path == "/api/v1/vm/vm-123/commit":
			w.WriteHeader(http.StatusCreated)
			w.Write([]byte(`{"commit_id":"commit-abc"}`))

		case r.Method == http.MethodPatch && r.URL.Path == "/api/v1/commits/commit-abc":
			body, _ := io.ReadAll(r.Body)
			json.Unmarshal(body, &patchBody)
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`{
				"commit_id": "commit-abc",
				"name": "just-a-name",
				"owner_id": "owner-1",
				"created_at": "2026-01-01T00:00:00Z",
				"is_public": false
			}`))

		default:
			t.Errorf("unexpected request: %s %s", r.Method, r.URL.Path)
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer server.Close()

	a := testApp(server.URL)
	res, err := handlers.HandleCommitCreate(context.Background(), a, handlers.CommitCreateReq{
		Target: "vm-123",
		Name:   "just-a-name",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if res.Name != "just-a-name" {
		t.Errorf("expected name just-a-name, got %s", res.Name)
	}
	if res.Description != "" {
		t.Errorf("expected empty description, got %s", res.Description)
	}

	// Verify name was sent but description was not
	if patchBody["name"] != "just-a-name" {
		t.Errorf("expected name in PATCH body, got %v", patchBody["name"])
	}
	if _, hasDesc := patchBody["description"]; hasDesc {
		t.Error("description should not be in PATCH body when not provided")
	}
}

func TestHandleCommitCreate_UpdateFails(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		switch {
		case r.Method == http.MethodGet && r.URL.Path == "/api/v1/vm/vm-123/status":
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`{"vm_id":"vm-123","owner_id":"owner-1","created_at":"2026-01-01T00:00:00Z","state":"running"}`))

		case r.Method == http.MethodPost && r.URL.Path == "/api/v1/vm/vm-123/commit":
			w.WriteHeader(http.StatusCreated)
			w.Write([]byte(`{"commit_id":"commit-abc"}`))

		case r.Method == http.MethodPatch:
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte(`{"error":"internal server error"}`))

		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer server.Close()

	a := testApp(server.URL)
	res, err := handlers.HandleCommitCreate(context.Background(), a, handlers.CommitCreateReq{
		Target: "vm-123",
		Name:   "my-commit",
	})

	// Should return an error but still have the commit ID
	if err == nil {
		t.Fatal("expected error when update fails")
	}
	if res.CommitID != "commit-abc" {
		t.Errorf("expected commit ID to be returned even on update failure, got %s", res.CommitID)
	}
}
