package handlers_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/hdresearch/vers-cli/internal/app"
	"github.com/hdresearch/vers-cli/internal/handlers"
	vers "github.com/hdresearch/vers-sdk-go"
	"github.com/hdresearch/vers-sdk-go/option"
)

func testApp(baseURL string) *app.App {
	client := vers.NewClient(
		option.WithBaseURL(baseURL),
		option.WithAPIKey("test-key"),
	)
	return &app.App{Client: client}
}

func TestHandleCommitsList(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v1/commits" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		if r.Method != http.MethodGet {
			t.Errorf("unexpected method: %s", r.Method)
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{
			"commits": [
				{
					"commit_id": "abc-123",
					"name": "my-commit",
					"owner_id": "owner-1",
					"created_at": "2026-03-17T00:00:00Z",
					"is_public": false
				}
			],
			"total": 1,
			"limit": 50,
			"offset": 0
		}`))
	}))
	defer server.Close()

	a := testApp(server.URL)
	res, err := handlers.HandleCommitsList(context.Background(), a, handlers.CommitsListReq{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(res.Commits) != 1 {
		t.Fatalf("expected 1 commit, got %d", len(res.Commits))
	}
	if res.Commits[0].CommitID != "abc-123" {
		t.Errorf("expected commit ID abc-123, got %s", res.Commits[0].CommitID)
	}
	if res.Commits[0].Name != "my-commit" {
		t.Errorf("expected name my-commit, got %s", res.Commits[0].Name)
	}
	if res.Total != 1 {
		t.Errorf("expected total 1, got %d", res.Total)
	}
	if res.Public {
		t.Error("expected Public=false")
	}
}

func TestHandleCommitsListPublic(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v1/commits/public" {
			t.Errorf("expected /api/v1/commits/public, got %s", r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{
			"commits": [],
			"total": 0,
			"limit": 50,
			"offset": 0
		}`))
	}))
	defer server.Close()

	a := testApp(server.URL)
	res, err := handlers.HandleCommitsList(context.Background(), a, handlers.CommitsListReq{Public: true})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !res.Public {
		t.Error("expected Public=true")
	}
	if len(res.Commits) != 0 {
		t.Errorf("expected 0 commits, got %d", len(res.Commits))
	}
}

func TestHandleCommitDelete(t *testing.T) {
	var deletedID string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodDelete {
			t.Errorf("expected DELETE, got %s", r.Method)
		}
		// Path is /api/v1/commits/{commit_id}
		deletedID = r.URL.Path[len("/api/v1/commits/"):]
		w.WriteHeader(http.StatusNoContent)
	}))
	defer server.Close()

	a := testApp(server.URL)

	// Test with empty commit ID
	err := handlers.HandleCommitDelete(context.Background(), a, handlers.CommitDeleteReq{})
	if err == nil {
		t.Fatal("expected error for empty commit ID")
	}

	// Test successful delete
	err = handlers.HandleCommitDelete(context.Background(), a, handlers.CommitDeleteReq{CommitID: "abc-123"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if deletedID != "abc-123" {
		t.Errorf("expected delete of abc-123, got %s", deletedID)
	}
}

func TestHandleCommitUpdate(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPatch {
			t.Errorf("expected PATCH, got %s", r.Method)
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{
			"commit_id": "abc-123",
			"name": "my-commit",
			"owner_id": "owner-1",
			"created_at": "2026-03-17T00:00:00Z",
			"is_public": true
		}`))
	}))
	defer server.Close()

	a := testApp(server.URL)

	// Test with empty commit ID
	_, err := handlers.HandleCommitUpdate(context.Background(), a, handlers.CommitUpdateReq{})
	if err == nil {
		t.Fatal("expected error for empty commit ID")
	}

	// Test successful update
	info, err := handlers.HandleCommitUpdate(context.Background(), a, handlers.CommitUpdateReq{
		CommitID: "abc-123",
		IsPublic: true,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if info.CommitID != "abc-123" {
		t.Errorf("expected commit ID abc-123, got %s", info.CommitID)
	}
	if !info.IsPublic {
		t.Error("expected IsPublic=true")
	}
}

func TestHandleCommitParents(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Errorf("expected GET, got %s", r.Method)
		}
		expectedPath := "/api/v1/vm/commits/abc-123/parents"
		if r.URL.Path != expectedPath {
			t.Errorf("expected path %s, got %s", expectedPath, r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`[
			{
				"id": "parent-1",
				"name": "parent-commit",
				"owner_id": "owner-1",
				"created_at": "2026-03-16T00:00:00Z",
				"is_public": false
			}
		]`))
	}))
	defer server.Close()

	a := testApp(server.URL)

	// Test with empty commit ID
	_, err := handlers.HandleCommitParents(context.Background(), a, handlers.CommitParentsReq{})
	if err == nil {
		t.Fatal("expected error for empty commit ID")
	}

	// Test successful list parents
	res, err := handlers.HandleCommitParents(context.Background(), a, handlers.CommitParentsReq{CommitID: "abc-123"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if res.CommitID != "abc-123" {
		t.Errorf("expected commit ID abc-123, got %s", res.CommitID)
	}
	if len(res.Parents) != 1 {
		t.Fatalf("expected 1 parent, got %d", len(res.Parents))
	}
	if res.Parents[0].ID != "parent-1" {
		t.Errorf("expected parent ID parent-1, got %s", res.Parents[0].ID)
	}
}

func TestHandleCommitDelete_APIError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusConflict)
		w.Write([]byte(`{"error": "commit has active VMs"}`))
	}))
	defer server.Close()

	a := testApp(server.URL)
	err := handlers.HandleCommitDelete(context.Background(), a, handlers.CommitDeleteReq{CommitID: "abc-123"})
	if err == nil {
		t.Fatal("expected error for conflict response")
	}
}
