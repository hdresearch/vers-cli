package cmd

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"strings"
	"testing"
)

// setAPIKeyEnv sets VERS_API_KEY for the duration of a subtest and restores
// the prior value on cleanup. Empty string means "no API key available" —
// in that case we also isolate HOME to a fresh tempdir so auth.GetAPIKey()
// cannot fall back to the developer's real ~/.versrc.
func setAPIKeyEnv(t *testing.T, value string) {
	t.Helper()
	priorKey, hadKey := os.LookupEnv("VERS_API_KEY")
	if err := os.Setenv("VERS_API_KEY", value); err != nil {
		t.Fatalf("setenv VERS_API_KEY: %v", err)
	}

	var priorHome string
	var hadHome bool
	if value == "" {
		// Isolate HOME so LoadConfig() can't find ~/.versrc on the dev box.
		priorHome, hadHome = os.LookupEnv("HOME")
		tempHome := t.TempDir()
		if err := os.Setenv("HOME", tempHome); err != nil {
			t.Fatalf("setenv HOME: %v", err)
		}
	}

	t.Cleanup(func() {
		if hadKey {
			os.Setenv("VERS_API_KEY", priorKey)
		} else {
			os.Unsetenv("VERS_API_KEY")
		}
		if value == "" {
			if hadHome {
				os.Setenv("HOME", priorHome)
			} else {
				os.Unsetenv("HOME")
			}
		}
	})
}

// mustParseURL is a small helper for test setup.
func mustParseURL(t *testing.T, s string) *url.URL {
	t.Helper()
	u, err := url.Parse(s)
	if err != nil {
		t.Fatalf("parse URL %q: %v", s, err)
	}
	return u
}

// TestLandingGetJSON_Success verifies happy-path behavior: correct path, GET
// method, Bearer auth header, Accept header, and JSON decoding.
func TestLandingGetJSON_Success(t *testing.T) {
	setAPIKeyEnv(t, "test-api-key-abc")

	var gotPath, gotMethod, gotAuth, gotAccept string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		gotMethod = r.Method
		gotAuth = r.Header.Get("Authorization")
		gotAccept = r.Header.Get("Accept")
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"url": "https://github.com/apps/x/installations/new?state=nonce-1"}`))
	}))
	defer server.Close()

	var resp installURLResponse
	code, err := landingGetJSON(context.Background(), mustParseURL(t, server.URL), "/api/github/install-url", &resp)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if code != http.StatusOK {
		t.Errorf("expected status 200, got %d", code)
	}
	if gotPath != "/api/github/install-url" {
		t.Errorf("expected path /api/github/install-url, got %s", gotPath)
	}
	if gotMethod != http.MethodGet {
		t.Errorf("expected GET, got %s", gotMethod)
	}
	if gotAuth != "Bearer test-api-key-abc" {
		t.Errorf("expected Bearer auth header, got %q", gotAuth)
	}
	if gotAccept != "application/json" {
		t.Errorf("expected Accept: application/json, got %q", gotAccept)
	}
	if !strings.Contains(resp.URL, "state=nonce-1") {
		t.Errorf("expected decoded url to contain state=nonce-1, got %q", resp.URL)
	}
}

// TestLandingGetJSON_NoAPIKey verifies we error out early with a clear
// message when no API key is available (no stranded HTTP request).
func TestLandingGetJSON_NoAPIKey(t *testing.T) {
	setAPIKeyEnv(t, "")

	requestFired := false
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestFired = true
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	_, err := landingGetJSON(context.Background(), mustParseURL(t, server.URL), "/whatever", nil)
	if err == nil {
		t.Fatal("expected error when VERS_API_KEY is unset, got nil")
	}
	if !strings.Contains(err.Error(), "authentication required") {
		t.Errorf("expected 'authentication required' in error, got %q", err.Error())
	}
	if requestFired {
		t.Error("expected no HTTP request to fire when API key missing")
	}
}

// TestLandingGetJSON_ServerError verifies non-2xx statuses surface as errors
// with the status code and response body exposed to the caller.
func TestLandingGetJSON_ServerError(t *testing.T) {
	setAPIKeyEnv(t, "k")

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusForbidden)
		w.Write([]byte(`{"error":"admin only"}`))
	}))
	defer server.Close()

	code, err := landingGetJSON(context.Background(), mustParseURL(t, server.URL), "/api/github/install-url", nil)
	if err == nil {
		t.Fatal("expected error for 403, got nil")
	}
	if code != http.StatusForbidden {
		t.Errorf("expected returned code 403, got %d", code)
	}
	if !strings.Contains(err.Error(), "403") {
		t.Errorf("expected error to mention status 403, got %q", err.Error())
	}
	if !strings.Contains(err.Error(), "admin only") {
		t.Errorf("expected error to include response body, got %q", err.Error())
	}
}

// TestLandingGetJSON_NumericInstallationID verifies that when the server
// sends installation_id as a JSON number (the corrected wire format after
// the vers-landing bigint→number coercion fix), the Go CLI decodes it
// cleanly into int64 without error.
func TestLandingGetJSON_NumericInstallationID(t *testing.T) {
	setAPIKeyEnv(t, "k")

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"installed":       true,
			"installation_id": int64(125660596),
			"org_id":          "abc",
			"org_name":        "acme",
		})
	}))
	defer server.Close()

	var status githubStatusResponse
	_, err := landingGetJSON(context.Background(), mustParseURL(t, server.URL), "/api/github/status", &status)
	if err != nil {
		t.Fatalf("unexpected decode error: %v", err)
	}
	if !status.Installed {
		t.Errorf("expected Installed=true, got %v", status.Installed)
	}
	if status.InstallationID != 125660596 {
		t.Errorf("expected InstallationID=125660596, got %d", status.InstallationID)
	}
	if status.Org != "" {
		// githubStatusResponse.Org only unmarshals if the server uses that key;
		// a legitimate response may use org_name instead, so this is soft.
	}
}

// TestLandingGetJSON_ContextCancel verifies cancellation propagates into
// the request (no hangs).
func TestLandingGetJSON_ContextCancel(t *testing.T) {
	setAPIKeyEnv(t, "k")

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Block long enough that the caller's cancelled context should win.
		<-r.Context().Done()
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // cancel immediately

	_, err := landingGetJSON(ctx, mustParseURL(t, server.URL), "/slow", nil)
	if err == nil {
		t.Fatal("expected error from cancelled context, got nil")
	}
}
