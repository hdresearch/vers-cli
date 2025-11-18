package test

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/hdresearch/vers-cli/internal/auth"
	"github.com/hdresearch/vers-cli/test/testutil"
	vers "github.com/hdresearch/vers-sdk-go"
)

// TestGetOrCreateSSHKey_ExistingKey tests that GetOrCreateSSHKey returns an existing key.
func TestGetOrCreateSSHKey_ExistingKey(t *testing.T) {
	t.Log("Starting TestGetOrCreateSSHKey_ExistingKey...")
	testutil.TestEnv(t)

	// Create a temporary directory for the test
	tempDir := t.TempDir()
	originalWd, err := os.Getwd()
	if err != nil {
		t.Fatalf("failed to get current directory: %v", err)
	}
	defer os.Chdir(originalWd)

	// Change to temp directory
	if err := os.Chdir(tempDir); err != nil {
		t.Fatalf("failed to change to temp directory: %v", err)
	}
	t.Logf("Working in temp directory: %s", tempDir)

	// Create the temp SSH keys directory structure
	keysDir := filepath.Join(os.TempDir(), "vers-ssh-keys")
	if err := os.MkdirAll(keysDir, 0755); err != nil {
		t.Fatalf("failed to create keys directory: %v", err)
	}

	// Create a dummy SSH key file
	testVMID := "test-vm-12345"
	keyPath := filepath.Join(keysDir, testVMID+".key")
	dummyKeyContent := `-----BEGIN OPENSSH PRIVATE KEY-----
b3BlbnNzaC1rZXktdjEAAAAABG5vbmUAAAAEbm9uZQAAAAAAAAABAAAAMwAAAAtz
c2gtZWQyNTUxOQAAACDummyJZD5xPH8cYmP5KkqLXQBLJ8aHcHXRqvNKwFdF9wAA
AIhwL2kBcC9pAQAAAAtzc2gtZWQyNTUxOQAAACDummyJZD5xPH8cYmP5KkqLXQBL
J8aHcHXRqvNKwFdF9wAAAEDummyJZD5xPH8cYmP5KkqLXQBLJ8aHcHXRqvNKwFdF
9wAAABHRlc3Rpa2V5QGV4YW1wbGUBAgM=
-----END OPENSSH PRIVATE KEY-----`

	if err := os.WriteFile(keyPath, []byte(dummyKeyContent), 0600); err != nil {
		t.Fatalf("failed to create dummy SSH key: %v", err)
	}
	t.Logf("Created dummy SSH key at: %s", keyPath)

	// Create a client (we don't need a real one since we're testing the cached key path)
	clientOptions, err := auth.GetClientOptions()
	if err != nil {
		t.Skipf("Could not get client options (may need API credentials): %v", err)
	}
	client := vers.NewClient(clientOptions...)

	// Call GetOrCreateSSHKey
	ctx := context.Background()
	returnedPath, err := auth.GetOrCreateSSHKey(testVMID, client, ctx)
	if err != nil {
		t.Fatalf("GetOrCreateSSHKey failed with existing key: %v", err)
	}

	t.Logf("Returned key path: %s", returnedPath)

	// Verify the returned path matches what we created
	if returnedPath != keyPath {
		t.Errorf("expected path %s, got %s", keyPath, returnedPath)
	}

	// Verify the key file still exists and has correct content
	content, err := os.ReadFile(returnedPath)
	if err != nil {
		t.Fatalf("failed to read returned key file: %v", err)
	}

	if string(content) != dummyKeyContent {
		t.Errorf("key file content doesn't match expected content")
	}

	t.Log("✓ GetOrCreateSSHKey successfully returned existing key path")
	t.Log("TestGetOrCreateSSHKey_ExistingKey completed")
}

// TestGetOrCreateSSHKey_NoKey tests that GetOrCreateSSHKey fetches and saves a new key.
func TestGetOrCreateSSHKey_NoKey(t *testing.T) {
	t.Log("Starting TestGetOrCreateSSHKey_NoKey...")
	testutil.TestEnv(t)

	// Create a temporary directory for the test
	tempDir := t.TempDir()
	originalWd, err := os.Getwd()
	if err != nil {
		t.Fatalf("failed to get current directory: %v", err)
	}
	defer os.Chdir(originalWd)

	// Change to temp directory
	if err := os.Chdir(tempDir); err != nil {
		t.Fatalf("failed to change to temp directory: %v", err)
	}
	t.Logf("Working in temp directory: %s", tempDir)

	// Don't create the keys directory - test when it doesn't exist
	testVMID := "nonexistent-vm-67890"

	// Create a client
	clientOptions, err := auth.GetClientOptions()
	if err != nil {
		t.Skipf("Could not get client options (may need API credentials): %v", err)
	}
	client := vers.NewClient(clientOptions...)

	// Call GetOrCreateSSHKey - should try to fetch from API
	ctx := context.Background()
	returnedPath, err := auth.GetOrCreateSSHKey(testVMID, client, ctx)

	// We expect an error because the VM doesn't exist
	if err == nil {
		t.Fatalf("expected error when VM doesn't exist, got nil. Returned path: %s", returnedPath)
	}

	t.Logf("Got expected error: %v", err)

	// Verify the error message is about failure to fetch
	if !strings.Contains(err.Error(), "failed to fetch") {
		t.Logf("Warning: error message doesn't mention 'failed to fetch'. Got: %s", err.Error())
	}

	t.Log("✓ GetOrCreateSSHKey correctly returns error when VM doesn't exist")
	t.Log("TestGetOrCreateSSHKey_NoKey completed")
}

// TestGetOrCreateSSHKey_Integration tests the full flow with a real VM.
func TestGetOrCreateSSHKey_Integration(t *testing.T) {
	t.Log("Starting TestGetOrCreateSSHKey_Integration...")
	testutil.TestEnv(t)
	testutil.EnsureBuilt(t)

	// Create a real VM (no need to change directories - testing API behavior)
	t.Log("Running: vers run")
	out, err := testutil.RunVers(t, testutil.DefaultTimeout, "run")
	if err != nil {
		t.Fatalf("vers run failed: %v\nOutput:\n%s", err, out)
	}

	// Parse VM ID
	vmID, err := testutil.ParseVMID(out)
	if err != nil {
		t.Fatalf("failed to parse VM ID from output: %v\nOutput:\n%s", err, out)
	}
	t.Logf("Created VM: %s", vmID)
	testutil.RegisterVMCleanup(t, vmID, false)

	// Get client
	clientOptions, err := auth.GetClientOptions()
	if err != nil {
		t.Fatalf("failed to get client options: %v", err)
	}
	client := vers.NewClient(clientOptions...)

	// Try to get SSH key for the real VM
	ctx := context.Background()
	keyPath, err := auth.GetOrCreateSSHKey(vmID, client, ctx)
	if err != nil {
		t.Fatalf("GetOrCreateSSHKey failed: %v", err)
	}

	t.Logf("SSH key path: %s", keyPath)

	// Verify the key file was created
	if _, statErr := os.Stat(keyPath); statErr != nil {
		t.Fatalf("SSH key file was not created at %s: %v", keyPath, statErr)
	}

	// Verify the key file has restrictive permissions (0600)
	fileInfo, err := os.Stat(keyPath)
	if err != nil {
		t.Fatalf("failed to stat key file: %v", err)
	}
	if fileInfo.Mode().Perm() != 0600 {
		t.Errorf("expected key file permissions 0600, got %o", fileInfo.Mode().Perm())
	}

	// Read the key file and verify it contains an SSH key
	keyContent, err := os.ReadFile(keyPath)
	if err != nil {
		t.Fatalf("failed to read key file: %v", err)
	}

	keyStr := string(keyContent)
	if !strings.Contains(keyStr, "BEGIN") || !strings.Contains(keyStr, "PRIVATE KEY") {
		t.Errorf("key file doesn't appear to contain a valid SSH private key. Content length: %d bytes", len(keyStr))
	}

	t.Logf("✓ SSH key successfully fetched and saved (%d bytes)", len(keyStr))

	// Test that calling it again returns the cached key
	t.Log("Testing cached key retrieval...")
	keyPath2, err := auth.GetOrCreateSSHKey(vmID, client, ctx)
	if err != nil {
		t.Fatalf("GetOrCreateSSHKey failed on second call: %v", err)
	}

	if keyPath2 != keyPath {
		t.Errorf("expected same key path on second call, got %s vs %s", keyPath, keyPath2)
	}

	t.Log("✓ Cached key retrieval works")
	t.Log("TestGetOrCreateSSHKey_Integration completed")
}
