package cmd

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
)

// setLandingURLEnv points the doMintToken resolver at our test server.
func setLandingURLEnv(t *testing.T, value string) {
	t.Helper()
	prior, had := os.LookupEnv("VERS_LANDING_URL")
	if err := os.Setenv("VERS_LANDING_URL", value); err != nil {
		t.Fatalf("setenv VERS_LANDING_URL: %v", err)
	}
	t.Cleanup(func() {
		if had {
			os.Setenv("VERS_LANDING_URL", prior)
		} else {
			os.Unsetenv("VERS_LANDING_URL")
		}
	})
}

// TestDoMintToken_Success verifies POST path, Bearer auth, Content-Type,
// request body shape (repositories + permissions), and decoded response.
func TestDoMintToken_Success(t *testing.T) {
	setAPIKeyEnv(t, "test-key")

	var gotPath, gotMethod, gotAuth, gotCT string
	var gotBody map[string]interface{}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		gotMethod = r.Method
		gotAuth = r.Header.Get("Authorization")
		gotCT = r.Header.Get("Content-Type")
		raw, _ := io.ReadAll(r.Body)
		_ = json.Unmarshal(raw, &gotBody)

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{
			"token": "ghs_abc123",
			"expires_at": "2026-04-20T22:15:46Z",
			"installation_id": 99999,
			"repository_selection": "selected",
			"permissions": {"contents": "read", "pull_requests": "write"}
		}`))
	}))
	defer server.Close()
	setLandingURLEnv(t, server.URL)

	resp, err := doMintToken(context.Background(), mintTokenRequest{
		Repositories: []string{"my-repo", "other-repo"},
		Permissions:  map[string]string{"contents": "read"},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.Token != "ghs_abc123" {
		t.Errorf("expected token ghs_abc123, got %s", resp.Token)
	}
	if resp.InstallationID != 99999 {
		t.Errorf("expected installation_id 99999, got %d", resp.InstallationID)
	}

	// Wire-format assertions
	if gotPath != "/api/github/installation-token" {
		t.Errorf("expected path /api/github/installation-token, got %s", gotPath)
	}
	if gotMethod != http.MethodPost {
		t.Errorf("expected POST, got %s", gotMethod)
	}
	if gotAuth != "Bearer test-key" {
		t.Errorf("expected Bearer auth, got %q", gotAuth)
	}
	if !strings.Contains(gotCT, "application/json") {
		t.Errorf("expected Content-Type application/json, got %q", gotCT)
	}

	// Body assertions
	repos, _ := gotBody["repositories"].([]interface{})
	if len(repos) != 2 || repos[0] != "my-repo" || repos[1] != "other-repo" {
		t.Errorf("expected repositories=[my-repo other-repo], got %v", repos)
	}
	perms, _ := gotBody["permissions"].(map[string]interface{})
	if perms["contents"] != "read" {
		t.Errorf("expected permissions.contents=read, got %v", perms["contents"])
	}
}

// TestDoMintToken_NoAPIKey verifies we error out early without firing a
// request when no API key is configured.
func TestDoMintToken_NoAPIKey(t *testing.T) {
	setAPIKeyEnv(t, "")

	requestFired := false
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestFired = true
	}))
	defer server.Close()
	setLandingURLEnv(t, server.URL)

	_, err := doMintToken(context.Background(), mintTokenRequest{})
	if err == nil {
		t.Fatal("expected error without API key")
	}
	if !strings.Contains(err.Error(), "authentication required") {
		t.Errorf("expected 'authentication required' in error, got %q", err.Error())
	}
	if requestFired {
		t.Error("expected no HTTP request to fire when API key missing")
	}
}

// TestDoMintToken_ErrorMapping verifies the status-code → friendly-message
// mapping (401/403 → auth failed, 404 → install-first nudge, 503 → app not
// configured, other 4xx/5xx → generic with code + body).
func TestDoMintToken_ErrorMapping(t *testing.T) {
	setAPIKeyEnv(t, "k")

	tests := []struct {
		name        string
		status      int
		body        string
		mustContain string
	}{
		{"401 auth", http.StatusUnauthorized, `{"error":"bad key"}`, "authentication failed"},
		{"403 auth", http.StatusForbidden, `{"error":"admin only"}`, "authentication failed"},
		{"404 install-first", http.StatusNotFound, `{"error":"no install"}`, "vers github install"},
		{"503 app unconfigured", http.StatusServiceUnavailable, `{"error":"no app"}`, "not configured"},
		{"422 generic", http.StatusUnprocessableEntity, `{"error":"bad repo"}`, "HTTP 422"},
		{"500 generic", http.StatusInternalServerError, `{"error":"boom"}`, "HTTP 500"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(tc.status)
				w.Write([]byte(tc.body))
			}))
			defer server.Close()
			setLandingURLEnv(t, server.URL)

			_, err := doMintToken(context.Background(), mintTokenRequest{})
			if err == nil {
				t.Fatalf("expected error for status %d, got nil", tc.status)
			}
			if !strings.Contains(err.Error(), tc.mustContain) {
				t.Errorf("expected error to contain %q, got %q", tc.mustContain, err.Error())
			}
		})
	}
}

// TestDoMintToken_EmptyToken verifies that a 200 response with no token
// still produces an error (not a silent success with an empty string).
func TestDoMintToken_EmptyToken(t *testing.T) {
	setAPIKeyEnv(t, "k")

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"token": "", "installation_id": 1}`))
	}))
	defer server.Close()
	setLandingURLEnv(t, server.URL)

	_, err := doMintToken(context.Background(), mintTokenRequest{})
	if err == nil {
		t.Fatal("expected error when API returns empty token, got nil")
	}
	if !strings.Contains(err.Error(), "empty token") {
		t.Errorf("expected 'empty token' in error, got %q", err.Error())
	}
}

// TestDoMintToken_UsesLandingURL verifies that VERS_LANDING_URL actually
// drives endpoint selection (regression guard against re-introducing the
// old host-munging heuristic).
func TestDoMintToken_UsesLandingURL(t *testing.T) {
	setAPIKeyEnv(t, "k")

	var hitHost string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		hitHost = r.Host
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"token":"ghs_x","installation_id":1}`))
	}))
	defer server.Close()
	setLandingURLEnv(t, server.URL)

	_, err := doMintToken(context.Background(), mintTokenRequest{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// Sanity: we hit the test server, not vers.sh.
	if strings.Contains(hitHost, "vers.sh") {
		t.Errorf("expected test server host, got %s (env var not honored)", hitHost)
	}
}
