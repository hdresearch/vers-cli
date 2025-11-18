package test

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/hdresearch/vers-cli/test/testutil"
)

// TestLoginWithValidKey tests logging in with a valid API key
func TestLoginWithValidKey(t *testing.T) {
	t.Log("Starting TestLoginWithValidKey...")
	testutil.TestEnv(t)
	testutil.EnsureBuilt(t)

	// Get the API key from environment
	apiKey := os.Getenv("VERS_API_KEY")
	if apiKey == "" {
		t.Skip("VERS_API_KEY not set")
	}

	// Create a temporary home directory for this test
	tempHome := t.TempDir()
	originalHome := os.Getenv("HOME")
	os.Setenv("HOME", tempHome)
	defer os.Setenv("HOME", originalHome)

	// Clear VERS_API_KEY so login uses the saved key
	os.Unsetenv("VERS_API_KEY")

	// Run login with valid API key
	t.Log("Running: vers login --token <api-key>")
	out, err := testutil.RunVers(t, testutil.DefaultTimeout, "login", "--token", apiKey)
	if err != nil {
		t.Fatalf("vers login failed: %v\nOutput:\n%s", err, out)
	}

	// Check output
	if !strings.Contains(out, "API key validated successfully") {
		t.Fatalf("expected 'API key validated successfully' in output, got:\n%s", out)
	}
	if !strings.Contains(out, "Successfully authenticated with Vers") {
		t.Fatalf("expected 'Successfully authenticated with Vers' in output, got:\n%s", out)
	}

	// Verify the key was saved to ~/.versrc
	versrcPath := filepath.Join(tempHome, ".versrc")
	if _, err := os.Stat(versrcPath); os.IsNotExist(err) {
		t.Fatalf("~/.versrc was not created")
	}

	// Read and verify the saved key
	data, err := os.ReadFile(versrcPath)
	if err != nil {
		t.Fatalf("failed to read ~/.versrc: %v", err)
	}

	var config struct {
		APIKey string `json:"apiKey"`
	}
	if err := json.Unmarshal(data, &config); err != nil {
		t.Fatalf("failed to parse ~/.versrc: %v", err)
	}

	if config.APIKey != apiKey {
		t.Fatalf("saved API key doesn't match. Expected: %s, Got: %s", apiKey, config.APIKey)
	}

	// Verify file permissions (should be 0600)
	info, err := os.Stat(versrcPath)
	if err != nil {
		t.Fatalf("failed to stat ~/.versrc: %v", err)
	}
	perm := info.Mode().Perm()
	if perm != 0600 {
		t.Fatalf("expected file permissions 0600, got %o", perm)
	}

	t.Log("✓ Login with valid key successful")
	t.Log("TestLoginWithValidKey completed")
}

// TestLoginWithInvalidKey tests logging in with an invalid API key
func TestLoginWithInvalidKey(t *testing.T) {
	t.Log("Starting TestLoginWithInvalidKey...")
	testutil.TestEnv(t)
	testutil.EnsureBuilt(t)

	// Create a temporary home directory for this test
	tempHome := t.TempDir()
	originalHome := os.Getenv("HOME")
	os.Setenv("HOME", tempHome)
	defer os.Setenv("HOME", originalHome)

	// Try to login with an invalid API key
	invalidKey := "invalid-api-key-12345"
	t.Log("Running: vers login --token <invalid-key>")
	out, err := testutil.RunVers(t, testutil.DefaultTimeout, "login", "--token", invalidKey)

	// Should fail
	if err == nil {
		t.Fatalf("expected login to fail with invalid key, but it succeeded. Output:\n%s", out)
	}

	// Check error message
	if !strings.Contains(out, "invalid API key") && !strings.Contains(out, "Forbidden") {
		t.Logf("Warning: expected 'invalid API key' or 'Forbidden' in error message, got:\n%s", out)
	}

	// Verify the key was NOT saved
	versrcPath := filepath.Join(tempHome, ".versrc")
	data, err := os.ReadFile(versrcPath)
	if err == nil && len(data) > 0 {
		var config struct {
			APIKey string `json:"apiKey"`
		}
		if json.Unmarshal(data, &config) == nil && config.APIKey == invalidKey {
			t.Fatalf("invalid API key should not have been saved to ~/.versrc")
		}
	}

	t.Log("✓ Login correctly rejected invalid key")
	t.Log("TestLoginWithInvalidKey completed")
}

// TestLoginAndUseCommands tests that commands work after login
func TestLoginAndUseCommands(t *testing.T) {
	t.Log("Starting TestLoginAndUseCommands...")
	testutil.TestEnv(t)
	testutil.EnsureBuilt(t)

	// Get the API key from environment
	apiKey := os.Getenv("VERS_API_KEY")
	if apiKey == "" {
		t.Skip("VERS_API_KEY not set")
	}

	// Create a temporary home directory for this test
	tempHome := t.TempDir()
	originalHome := os.Getenv("HOME")
	os.Setenv("HOME", tempHome)
	defer os.Setenv("HOME", originalHome)

	// Login
	t.Log("Running: vers login --token <api-key>")
	out, err := testutil.RunVers(t, testutil.DefaultTimeout, "login", "--token", apiKey)
	if err != nil {
		t.Fatalf("vers login failed: %v\nOutput:\n%s", err, out)
	}

	// Clear VERS_API_KEY so commands use the saved key
	os.Unsetenv("VERS_API_KEY")

	// Try to run status command (should work with saved key)
	t.Log("Running: vers status (using saved key)")
	out, err = testutil.RunVers(t, testutil.DefaultTimeout, "status")
	if err != nil {
		t.Fatalf("vers status failed after login: %v\nOutput:\n%s", err, out)
	}

	// Should show VMs or at least not fail with auth error
	if strings.Contains(strings.ToLower(out), "unauthorized") || strings.Contains(strings.ToLower(out), "please run 'vers login'") {
		t.Fatalf("status command failed with auth error after login. Output:\n%s", out)
	}

	t.Log("✓ Commands work with saved API key")
	t.Log("TestLoginAndUseCommands completed")
}

// TestLoginOverwritesExistingKey tests that login overwrites an existing key
func TestLoginOverwritesExistingKey(t *testing.T) {
	t.Log("Starting TestLoginOverwritesExistingKey...")
	testutil.TestEnv(t)
	testutil.EnsureBuilt(t)

	// Get the API key from environment
	apiKey := os.Getenv("VERS_API_KEY")
	if apiKey == "" {
		t.Skip("VERS_API_KEY not set")
	}

	// Create a temporary home directory for this test
	tempHome := t.TempDir()
	originalHome := os.Getenv("HOME")
	os.Setenv("HOME", tempHome)
	defer os.Setenv("HOME", originalHome)

	// Create a fake existing .versrc with a different key
	versrcPath := filepath.Join(tempHome, ".versrc")
	oldKey := "old-fake-key-12345"
	oldConfig := map[string]string{"apiKey": oldKey}
	oldData, _ := json.MarshalIndent(oldConfig, "", "  ")
	if err := os.WriteFile(versrcPath, oldData, 0600); err != nil {
		t.Fatalf("failed to create fake ~/.versrc: %v", err)
	}

	// Login with new key
	t.Log("Running: vers login --token <new-api-key>")
	out, err := testutil.RunVers(t, testutil.DefaultTimeout, "login", "--token", apiKey)
	if err != nil {
		t.Fatalf("vers login failed: %v\nOutput:\n%s", err, out)
	}

	// Verify the key was updated
	data, err := os.ReadFile(versrcPath)
	if err != nil {
		t.Fatalf("failed to read ~/.versrc: %v", err)
	}

	var config struct {
		APIKey string `json:"apiKey"`
	}
	if err := json.Unmarshal(data, &config); err != nil {
		t.Fatalf("failed to parse ~/.versrc: %v", err)
	}

	if config.APIKey == oldKey {
		t.Fatalf("API key was not updated. Still has old key: %s", oldKey)
	}
	if config.APIKey != apiKey {
		t.Fatalf("API key doesn't match new key. Expected: %s, Got: %s", apiKey, config.APIKey)
	}

	t.Log("✓ Login correctly overwrites existing key")
	t.Log("TestLoginOverwritesExistingKey completed")
}
