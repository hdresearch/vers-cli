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

func TestHandleResize(t *testing.T) {
	var receivedBody map[string]interface{}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch {
		case r.URL.Path == "/api/v1/vm/vm-123/status" && r.Method == http.MethodGet:
			// ResolveVMIdentifier calls Status
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`{"vm_id": "vm-123", "owner_id": "owner-1", "created_at": "2026-03-17T00:00:00Z", "state": "Running"}`))
		case r.URL.Path == "/api/v1/vm/vm-123/disk" && r.Method == http.MethodPatch:
			body, _ := io.ReadAll(r.Body)
			json.Unmarshal(body, &receivedBody)
			w.WriteHeader(http.StatusNoContent)
		default:
			t.Errorf("unexpected request: %s %s", r.Method, r.URL.Path)
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer server.Close()

	a := testApp(server.URL)

	// Test with invalid size
	_, err := handlers.HandleResize(context.Background(), a, handlers.ResizeReq{
		Target:    "vm-123",
		FsSizeMib: 0,
	})
	if err == nil {
		t.Fatal("expected error for zero size")
	}

	_, err = handlers.HandleResize(context.Background(), a, handlers.ResizeReq{
		Target:    "vm-123",
		FsSizeMib: -100,
	})
	if err == nil {
		t.Fatal("expected error for negative size")
	}

	// Test successful resize
	vmID, err := handlers.HandleResize(context.Background(), a, handlers.ResizeReq{
		Target:    "vm-123",
		FsSizeMib: 20480,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if vmID != "vm-123" {
		t.Errorf("expected VM ID vm-123, got %s", vmID)
	}

	// Verify request body
	if receivedBody["fs_size_mib"] != float64(20480) {
		t.Errorf("expected fs_size_mib=20480, got %v", receivedBody["fs_size_mib"])
	}
}

func TestHandleResize_APIError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch {
		case r.URL.Path == "/api/v1/vm/vm-123/status":
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`{"vm_id": "vm-123", "owner_id": "owner-1", "created_at": "2026-03-17T00:00:00Z", "state": "Running"}`))
		default:
			w.WriteHeader(http.StatusBadRequest)
			w.Write([]byte(`{"error": "new size must be greater than current size"}`))
		}
	}))
	defer server.Close()

	a := testApp(server.URL)
	_, err := handlers.HandleResize(context.Background(), a, handlers.ResizeReq{
		Target:    "vm-123",
		FsSizeMib: 1024,
	})
	if err == nil {
		t.Fatal("expected error for bad request")
	}
}
