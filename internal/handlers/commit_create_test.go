package handlers_test

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/hdresearch/vers-cli/internal/app"
	"github.com/hdresearch/vers-cli/internal/handlers"
	vers "github.com/hdresearch/vers-sdk-go"
	"github.com/hdresearch/vers-sdk-go/option"
)

// testAppWithStderr returns an App pointed at baseURL with a captured
// stderr writer, for testing handler warnings.
func testAppWithStderr(baseURL string, stderr io.Writer) *app.App {
	client := vers.NewClient(
		option.WithBaseURL(baseURL),
		option.WithAPIKey("test-key"),
	)
	return &app.App{
		Client: client,
		IO:     app.Output{In: nil, Out: io.Discard, Err: stderr},
	}
}

func TestHandleCommitCreate_WithName(t *testing.T) {
	var commitBody map[string]interface{}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		switch {
		case r.Method == http.MethodGet && r.URL.Path == "/api/v1/vm/vm-123/status":
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`{"vm_id":"vm-123","owner_id":"owner-1","created_at":"2026-01-01T00:00:00Z","state":"running"}`))

		case r.Method == http.MethodPost && r.URL.Path == "/api/v1/vm/vm-123/commit":
			body, _ := io.ReadAll(r.Body)
			json.Unmarshal(body, &commitBody)
			w.WriteHeader(http.StatusCreated)
			w.Write([]byte(`{"commit_id":"commit-abc"}`))

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

	// Verify name and description were sent in the commit request body
	if commitBody == nil {
		t.Fatal("expected commit request to have a body")
	}
	if commitBody["name"] != "my-commit" {
		t.Errorf("expected body name=my-commit, got %v", commitBody["name"])
	}
	if commitBody["description"] != "my description" {
		t.Errorf("expected body description='my description', got %v", commitBody["description"])
	}
}

func TestHandleCommitCreate_WithoutName(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		switch {
		case r.Method == http.MethodGet && r.URL.Path == "/api/v1/vm/vm-123/status":
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`{"vm_id":"vm-123","owner_id":"owner-1","created_at":"2026-01-01T00:00:00Z","state":"running"}`))

		case r.Method == http.MethodPost && r.URL.Path == "/api/v1/vm/vm-123/commit":
			w.WriteHeader(http.StatusCreated)
			w.Write([]byte(`{"commit_id":"commit-abc"}`))

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
	if res.Name != "" {
		t.Errorf("expected empty name, got %s", res.Name)
	}
}

func TestHandleCommitCreate_NameOnly(t *testing.T) {
	var commitBody map[string]interface{}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		switch {
		case r.Method == http.MethodGet && r.URL.Path == "/api/v1/vm/vm-123/status":
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`{"vm_id":"vm-123","owner_id":"owner-1","created_at":"2026-01-01T00:00:00Z","state":"running"}`))

		case r.Method == http.MethodPost && r.URL.Path == "/api/v1/vm/vm-123/commit":
			body, _ := io.ReadAll(r.Body)
			json.Unmarshal(body, &commitBody)
			w.WriteHeader(http.StatusCreated)
			w.Write([]byte(`{"commit_id":"commit-abc"}`))

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
	if commitBody["name"] != "just-a-name" {
		t.Errorf("expected name in body, got %v", commitBody["name"])
	}
	if _, hasDesc := commitBody["description"]; hasDesc {
		t.Error("description should not be in body when not provided")
	}
}

// ── --tag + --public tests ───────────────────────────────────────────

func TestHandleCommitCreate_TagBadFormat(t *testing.T) {
	// No server should be hit: validation runs before any side effect.
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Errorf("unexpected request: %s %s", r.Method, r.URL.Path)
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()

	a := testApp(server.URL)
	_, err := handlers.HandleCommitCreate(context.Background(), a, handlers.CommitCreateReq{
		Target: "vm-123",
		Tags:   []string{"v1.2"},
	})
	if err == nil {
		t.Fatal("expected error for bad --tag format")
	}
	want := `--tag must be in <repo>:<tag> form (got: "v1.2")`
	if !strings.Contains(err.Error(), want) {
		t.Errorf("expected error to contain %q, got %q", want, err.Error())
	}
}

func TestHandleCommitCreate_TagRepoNotFound(t *testing.T) {
	var commitCalled bool
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch {
		case r.Method == http.MethodGet && r.URL.Path == "/api/v1/repositories/nope":
			w.WriteHeader(http.StatusNotFound)
			w.Write([]byte(`{"error":"not found"}`))
		case r.URL.Path == "/api/v1/vm/vm-123/commit":
			commitCalled = true
			w.WriteHeader(http.StatusCreated)
			w.Write([]byte(`{"commit_id":"commit-abc"}`))
		default:
			t.Errorf("unexpected request: %s %s", r.Method, r.URL.Path)
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer server.Close()

	a := testApp(server.URL)
	_, err := handlers.HandleCommitCreate(context.Background(), a, handlers.CommitCreateReq{
		Target: "vm-123",
		Tags:   []string{"nope:v1"},
	})
	if err == nil {
		t.Fatal("expected error for missing repo")
	}
	want := `repo "nope" not found. Create it first with: vers repo create nope`
	if !strings.Contains(err.Error(), want) {
		t.Errorf("expected error to contain %q, got %q", want, err.Error())
	}
	if commitCalled {
		t.Error("commit endpoint must not be called when repo lookup fails")
	}
}

func TestHandleCommitCreate_TagCreateNew(t *testing.T) {
	var createTagBody map[string]interface{}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch {
		case r.Method == http.MethodGet && r.URL.Path == "/api/v1/repositories/my-app":
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`{"name":"my-app","repo_id":"repo-1","is_public":false,"created_at":"2026-01-01T00:00:00Z","description":null}`))
		case r.Method == http.MethodGet && r.URL.Path == "/api/v1/vm/vm-123/status":
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`{"vm_id":"vm-123","owner_id":"owner-1","created_at":"2026-01-01T00:00:00Z","state":"running"}`))
		case r.Method == http.MethodPost && r.URL.Path == "/api/v1/vm/vm-123/commit":
			w.WriteHeader(http.StatusCreated)
			w.Write([]byte(`{"commit_id":"commit-abc"}`))
		case r.Method == http.MethodGet && r.URL.Path == "/api/v1/repositories/my-app/tags/v1":
			w.WriteHeader(http.StatusNotFound)
			w.Write([]byte(`{"error":"not found"}`))
		case r.Method == http.MethodPost && r.URL.Path == "/api/v1/repositories/my-app/tags":
			body, _ := io.ReadAll(r.Body)
			json.Unmarshal(body, &createTagBody)
			w.WriteHeader(http.StatusCreated)
			w.Write([]byte(`{"commit_id":"commit-abc","reference":"my-app:v1","tag_id":"tag-new"}`))
		default:
			t.Errorf("unexpected request: %s %s", r.Method, r.URL.Path)
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer server.Close()

	a := testApp(server.URL)
	res, err := handlers.HandleCommitCreate(context.Background(), a, handlers.CommitCreateReq{
		Target: "vm-123",
		Tags:   []string{"my-app:v1"},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(res.TagsWritten) != 1 {
		t.Fatalf("expected 1 tag written, got %d", len(res.TagsWritten))
	}
	got := res.TagsWritten[0]
	if got.Reference != "my-app:v1" || got.TagID != "tag-new" {
		t.Errorf("unexpected tag written: %+v", got)
	}
	if createTagBody["tag_name"] != "v1" || createTagBody["commit_id"] != "commit-abc" {
		t.Errorf("unexpected create-tag body: %+v", createTagBody)
	}
	if res.IsPublic {
		t.Error("expected IsPublic=false when --public not supplied")
	}
}

func TestHandleCommitCreate_TagUpdateExisting(t *testing.T) {
	var updateBody map[string]interface{}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch {
		case r.Method == http.MethodGet && r.URL.Path == "/api/v1/repositories/my-app":
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`{"name":"my-app","repo_id":"repo-1","is_public":false,"created_at":"2026-01-01T00:00:00Z","description":null}`))
		case r.Method == http.MethodGet && r.URL.Path == "/api/v1/vm/vm-123/status":
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`{"vm_id":"vm-123","owner_id":"owner-1","created_at":"2026-01-01T00:00:00Z","state":"running"}`))
		case r.Method == http.MethodPost && r.URL.Path == "/api/v1/vm/vm-123/commit":
			w.WriteHeader(http.StatusCreated)
			w.Write([]byte(`{"commit_id":"commit-new"}`))
		case r.Method == http.MethodGet && r.URL.Path == "/api/v1/repositories/my-app/tags/latest":
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`{"commit_id":"commit-old","reference":"my-app:latest","tag_id":"tag-existing","tag_name":"latest","created_at":"2026-01-01T00:00:00Z","updated_at":"2026-01-01T00:00:00Z","description":null}`))
		case r.Method == http.MethodPatch && r.URL.Path == "/api/v1/repositories/my-app/tags/latest":
			body, _ := io.ReadAll(r.Body)
			json.Unmarshal(body, &updateBody)
			w.WriteHeader(http.StatusNoContent)
		default:
			t.Errorf("unexpected request: %s %s", r.Method, r.URL.Path)
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer server.Close()

	a := testApp(server.URL)
	res, err := handlers.HandleCommitCreate(context.Background(), a, handlers.CommitCreateReq{
		Target: "vm-123",
		Tags:   []string{"my-app:latest"},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(res.TagsWritten) != 1 || res.TagsWritten[0].TagID != "tag-existing" {
		t.Fatalf("expected reuse of existing tag_id, got %+v", res.TagsWritten)
	}
	if updateBody["commit_id"] != "commit-new" {
		t.Errorf("expected PATCH to set commit_id=commit-new, got %+v", updateBody)
	}
}

func TestHandleCommitCreate_PublicTriggersPublish(t *testing.T) {
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
			w.Write([]byte(`{"commit_id":"commit-abc","name":"","owner_id":"owner-1","created_at":"2026-01-01T00:00:00Z","is_public":true}`))
		default:
			t.Errorf("unexpected request: %s %s", r.Method, r.URL.Path)
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer server.Close()

	a := testApp(server.URL)
	res, err := handlers.HandleCommitCreate(context.Background(), a, handlers.CommitCreateReq{
		Target: "vm-123",
		Public: true,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !res.IsPublic {
		t.Error("expected IsPublic=true after --public")
	}
	if patchBody["is_public"] != true {
		t.Errorf("expected publish PATCH with is_public=true, got %+v", patchBody)
	}
}

func TestHandleCommitCreate_PublicRepoWarning(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch {
		case r.Method == http.MethodGet && r.URL.Path == "/api/v1/repositories/my-app":
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`{"name":"my-app","repo_id":"repo-1","is_public":true,"created_at":"2026-01-01T00:00:00Z","description":null}`))
		case r.Method == http.MethodGet && r.URL.Path == "/api/v1/vm/vm-123/status":
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`{"vm_id":"vm-123","owner_id":"owner-1","created_at":"2026-01-01T00:00:00Z","state":"running"}`))
		case r.Method == http.MethodPost && r.URL.Path == "/api/v1/vm/vm-123/commit":
			w.WriteHeader(http.StatusCreated)
			w.Write([]byte(`{"commit_id":"commit-abc"}`))
		case r.Method == http.MethodGet && r.URL.Path == "/api/v1/repositories/my-app/tags/v1":
			w.WriteHeader(http.StatusNotFound)
			w.Write([]byte(`{"error":"not found"}`))
		case r.Method == http.MethodPost && r.URL.Path == "/api/v1/repositories/my-app/tags":
			w.WriteHeader(http.StatusCreated)
			w.Write([]byte(`{"commit_id":"commit-abc","reference":"my-app:v1","tag_id":"tag-new"}`))
		default:
			t.Errorf("unexpected request: %s %s", r.Method, r.URL.Path)
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer server.Close()

	var stderr bytes.Buffer
	a := testAppWithStderr(server.URL, &stderr)
	_, err := handlers.HandleCommitCreate(context.Background(), a, handlers.CommitCreateReq{
		Target: "vm-123",
		Tags:   []string{"my-app:v1"},
		// no --public
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(stderr.String(), "warning:") || !strings.Contains(stderr.String(), "my-app") || !strings.Contains(stderr.String(), "--public") {
		t.Errorf("expected stderr to contain visibility warning, got %q", stderr.String())
	}
}
