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

func TestHandleTagCreate(t *testing.T) {
	var receivedBody map[string]interface{}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v1/commit_tags" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		if r.Method != http.MethodPost {
			t.Errorf("expected POST, got %s", r.Method)
		}
		body, _ := io.ReadAll(r.Body)
		json.Unmarshal(body, &receivedBody)

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		w.Write([]byte(`{
			"tag_id": "tag-uuid-1",
			"tag_name": "production",
			"commit_id": "abc-123"
		}`))
	}))
	defer server.Close()

	a := testApp(server.URL)

	// Test with empty tag name
	_, err := handlers.HandleTagCreate(context.Background(), a, handlers.TagCreateReq{})
	if err == nil {
		t.Fatal("expected error for empty tag name")
	}

	// Test with empty commit ID
	_, err = handlers.HandleTagCreate(context.Background(), a, handlers.TagCreateReq{TagName: "prod"})
	if err == nil {
		t.Fatal("expected error for empty commit ID")
	}

	// Test successful create
	resp, err := handlers.HandleTagCreate(context.Background(), a, handlers.TagCreateReq{
		TagName:     "production",
		CommitID:    "abc-123",
		Description: "production release",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.TagName != "production" {
		t.Errorf("expected tag name production, got %s", resp.TagName)
	}
	if resp.CommitID != "abc-123" {
		t.Errorf("expected commit ID abc-123, got %s", resp.CommitID)
	}

	// Verify the request body
	if receivedBody["tag_name"] != "production" {
		t.Errorf("expected tag_name=production in body, got %v", receivedBody["tag_name"])
	}
	if receivedBody["commit_id"] != "abc-123" {
		t.Errorf("expected commit_id=abc-123 in body, got %v", receivedBody["commit_id"])
	}
}

func TestHandleTagList(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v1/commit_tags" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		if r.Method != http.MethodGet {
			t.Errorf("expected GET, got %s", r.Method)
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{
			"tags": [
				{
					"tag_id": "tag-uuid-1",
					"tag_name": "production",
					"commit_id": "abc-123",
					"created_at": "2026-03-17T00:00:00Z",
					"updated_at": "2026-03-17T00:00:00Z"
				},
				{
					"tag_id": "tag-uuid-2",
					"tag_name": "staging",
					"commit_id": "def-456",
					"description": "staging env",
					"created_at": "2026-03-16T00:00:00Z",
					"updated_at": "2026-03-16T00:00:00Z"
				}
			]
		}`))
	}))
	defer server.Close()

	a := testApp(server.URL)
	res, err := handlers.HandleTagList(context.Background(), a, handlers.TagListReq{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(res.Tags) != 2 {
		t.Fatalf("expected 2 tags, got %d", len(res.Tags))
	}
	if res.Tags[0].TagName != "production" {
		t.Errorf("expected first tag production, got %s", res.Tags[0].TagName)
	}
	if res.Tags[1].Description != "staging env" {
		t.Errorf("expected description 'staging env', got %s", res.Tags[1].Description)
	}
}

func TestHandleTagListEmpty(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"tags": []}`))
	}))
	defer server.Close()

	a := testApp(server.URL)
	res, err := handlers.HandleTagList(context.Background(), a, handlers.TagListReq{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(res.Tags) != 0 {
		t.Errorf("expected 0 tags, got %d", len(res.Tags))
	}
}

func TestHandleTagGet(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		expectedPath := "/api/v1/commit_tags/production"
		if r.URL.Path != expectedPath {
			t.Errorf("expected path %s, got %s", expectedPath, r.URL.Path)
		}
		if r.Method != http.MethodGet {
			t.Errorf("expected GET, got %s", r.Method)
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{
			"tag_id": "tag-uuid-1",
			"tag_name": "production",
			"commit_id": "abc-123",
			"description": "prod release",
			"created_at": "2026-03-17T00:00:00Z",
			"updated_at": "2026-03-17T00:00:00Z"
		}`))
	}))
	defer server.Close()

	a := testApp(server.URL)

	// Test with empty tag name
	_, err := handlers.HandleTagGet(context.Background(), a, handlers.TagGetReq{})
	if err == nil {
		t.Fatal("expected error for empty tag name")
	}

	// Test successful get
	info, err := handlers.HandleTagGet(context.Background(), a, handlers.TagGetReq{TagName: "production"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if info.TagName != "production" {
		t.Errorf("expected tag name production, got %s", info.TagName)
	}
	if info.CommitID != "abc-123" {
		t.Errorf("expected commit ID abc-123, got %s", info.CommitID)
	}
	if info.Description != "prod release" {
		t.Errorf("expected description 'prod release', got %s", info.Description)
	}
}

func TestHandleTagUpdate(t *testing.T) {
	var receivedBody map[string]interface{}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		expectedPath := "/api/v1/commit_tags/production"
		if r.URL.Path != expectedPath {
			t.Errorf("expected path %s, got %s", expectedPath, r.URL.Path)
		}
		if r.Method != http.MethodPatch {
			t.Errorf("expected PATCH, got %s", r.Method)
		}
		body, _ := io.ReadAll(r.Body)
		json.Unmarshal(body, &receivedBody)
		w.WriteHeader(http.StatusNoContent)
	}))
	defer server.Close()

	a := testApp(server.URL)

	// Test with empty tag name
	err := handlers.HandleTagUpdate(context.Background(), a, handlers.TagUpdateReq{})
	if err == nil {
		t.Fatal("expected error for empty tag name")
	}

	// Test successful update with commit ID
	err = handlers.HandleTagUpdate(context.Background(), a, handlers.TagUpdateReq{
		TagName:  "production",
		CommitID: "new-commit-456",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if receivedBody["commit_id"] != "new-commit-456" {
		t.Errorf("expected commit_id=new-commit-456, got %v", receivedBody["commit_id"])
	}
}

func TestHandleTagDelete(t *testing.T) {
	var deletedTag string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodDelete {
			t.Errorf("expected DELETE, got %s", r.Method)
		}
		deletedTag = r.URL.Path[len("/api/v1/commit_tags/"):]
		w.WriteHeader(http.StatusNoContent)
	}))
	defer server.Close()

	a := testApp(server.URL)

	// Test with empty tag name
	err := handlers.HandleTagDelete(context.Background(), a, handlers.TagDeleteReq{})
	if err == nil {
		t.Fatal("expected error for empty tag name")
	}

	// Test successful delete
	err = handlers.HandleTagDelete(context.Background(), a, handlers.TagDeleteReq{TagName: "production"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if deletedTag != "production" {
		t.Errorf("expected delete of 'production', got %s", deletedTag)
	}
}

func TestHandleTagGet_NotFound(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusNotFound)
		w.Write([]byte(`{"error": "Tag not found"}`))
	}))
	defer server.Close()

	a := testApp(server.URL)
	_, err := handlers.HandleTagGet(context.Background(), a, handlers.TagGetReq{TagName: "nonexistent"})
	if err == nil {
		t.Fatal("expected error for not found tag")
	}
}

func TestHandleTagCreate_Conflict(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusConflict)
		w.Write([]byte(`{"error": "Tag name already exists in organization"}`))
	}))
	defer server.Close()

	a := testApp(server.URL)
	_, err := handlers.HandleTagCreate(context.Background(), a, handlers.TagCreateReq{
		TagName:  "production",
		CommitID: "abc-123",
	})
	if err == nil {
		t.Fatal("expected error for duplicate tag")
	}
}
