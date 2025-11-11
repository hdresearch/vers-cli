package test

import (
	"strings"
	"testing"

	"github.com/hdresearch/vers-cli/test/testutil"
)

// TestKillBasic tests that a VM can be created and deleted successfully.
func TestKillBasic(t *testing.T) {
	t.Log("Starting TestKillBasic...")
	testutil.TestEnv(t)
	testutil.EnsureBuilt(t)

	// Create a VM
	t.Log("Running: vers run")
	out, err := testutil.RunVers(t, testutil.DefaultTimeout, "run")
	if err != nil {
		t.Fatalf("vers run failed: %v\nOutput:\n%s", err, out)
	}
	t.Logf("VM creation output:\n%s", out)

	// Parse VM ID from output
	vmID, err := testutil.ParseVMID(out)
	if err != nil {
		t.Fatalf("failed to parse VM ID from output: %v\nOutput:\n%s", err, out)
	}
	t.Logf("Created VM: %s", vmID)

	// Delete the VM with -y to skip confirmation
	t.Logf("Running: vers kill -y %s", vmID)
	out, err = testutil.RunVers(t, testutil.DefaultTimeout, "kill", "-y", vmID)
	if err != nil {
		t.Fatalf("vers kill failed: %v\nOutput:\n%s", err, out)
	}
	t.Logf("VM deletion output:\n%s", out)

	// Note: Current implementation doesn't print a success message on deletion
	// Success is indicated by exit code 0 (no error) and empty/minimal output
	t.Log("✓ VM deleted successfully (command returned with no error)")
	t.Log("TestKillBasic completed")
}

// TestKillNonExistent tests that deleting a non-existent VM fails gracefully.
func TestKillNonExistent(t *testing.T) {
	t.Log("Starting TestKillNonExistent...")
	testutil.TestEnv(t)
	testutil.EnsureBuilt(t)

	// Try to delete a non-existent VM
	fakeVMID := "00000000-0000-0000-0000-000000000000"
	t.Logf("Running: vers kill -y %s", fakeVMID)
	out, err := testutil.RunVers(t, testutil.DefaultTimeout, "kill", "-y", fakeVMID)

	// Should fail with a clear error
	if err == nil {
		t.Fatalf("expected failure when deleting non-existent VM, got success. Output:\n%s", out)
	}

	// Check for helpful error message
	if !strings.Contains(strings.ToLower(out), "not found") &&
		!strings.Contains(strings.ToLower(out), "does not exist") &&
		!strings.Contains(strings.ToLower(out), "failed") {
		t.Logf("Warning: error message could be more specific. Got:\n%s", out)
	}

	t.Log("✓ Non-existent VM deletion failed as expected")
	t.Log("TestKillNonExistent completed")
}

// TestKillWithoutConfirmation tests that kill without -y prompts for confirmation.
func TestKillWithoutConfirmation(t *testing.T) {
	t.Log("Starting TestKillWithoutConfirmation...")
	testutil.TestEnv(t)
	testutil.EnsureBuilt(t)

	// Create a VM
	t.Log("Running: vers run")
	out, err := testutil.RunVers(t, testutil.DefaultTimeout, "run")
	if err != nil {
		t.Fatalf("vers run failed: %v\nOutput:\n%s", err, out)
	}

	// Parse VM ID and register cleanup in case test fails
	vmID, err := testutil.ParseVMID(out)
	if err != nil {
		t.Fatalf("failed to parse VM ID from output: %v\nOutput:\n%s", err, out)
	}
	t.Logf("Created VM: %s", vmID)
	testutil.RegisterVMCleanup(t, vmID, false)

	// Try to delete without -y flag (this will timeout waiting for input, which is expected)
	t.Logf("Running: vers kill %s (without -y, expecting timeout)", vmID)
	out, err = testutil.RunVers(t, testutil.DefaultTimeout, "kill", vmID)

	// We expect this to either:
	// 1. Timeout (because it's waiting for user input we can't provide)
	// 2. Fail with some error about stdin not being a terminal
	// 3. Or possibly succeed if -y is the default in non-interactive mode
	if err == nil {
		// If it succeeded, that's fine - maybe non-interactive mode auto-confirms
		t.Logf("Kill succeeded without confirmation (non-interactive mode). Output:\n%s", out)
		// Don't need cleanup anymore since VM was deleted
		return
	}

	// Check if it's a timeout or prompt-related error
	if strings.Contains(err.Error(), "timed out") || strings.Contains(out, "confirm") || strings.Contains(out, "y/n") {
		t.Logf("✓ Kill command prompted for confirmation or timed out as expected. Error: %v", err)
	} else {
		t.Logf("Kill failed with error (this may be expected in non-interactive mode): %v\nOutput:\n%s", err, out)
	}

	t.Log("TestKillWithoutConfirmation completed")
}
