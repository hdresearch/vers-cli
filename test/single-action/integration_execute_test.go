package test

import (
	"strings"
	"testing"
	"time"

	"github.com/hdresearch/vers-cli/test/testutil"
)

// TestExecuteBasic tests the basic `vers execute` command runs a command successfully.
func TestExecuteBasic(t *testing.T) {
	t.Log("Starting TestExecuteBasic...")
	testutil.TestEnv(t)
	testutil.EnsureBuilt(t)

	// Create a VM
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

	// Wait for VM to be fully ready
	t.Log("Waiting for VM networking to be configured...")
	if err := waitForVMReady(t, vmID, 60*time.Second); err != nil {
		t.Fatalf("VM did not become ready: %v", err)
	}

	// Execute a simple echo command
	t.Log("Running: vers execute", vmID, "echo", "hello-from-vers")
	out, err = testutil.RunVers(t, testutil.DefaultTimeout, "execute", vmID, "echo", "hello-from-vers")
	if err != nil {
		t.Fatalf("vers execute failed: %v\nOutput:\n%s", err, out)
	}

	// Verify output
	if !strings.Contains(out, "hello-from-vers") {
		t.Fatalf("expected 'hello-from-vers' in output, got:\n%s", out)
	}

	t.Log("✓ Execute command successful")
	t.Log("TestExecuteBasic completed")
}

// TestExecuteWithFlags tests execute command with flags that need -- separator.
func TestExecuteWithFlags(t *testing.T) {
	t.Log("Starting TestExecuteWithFlags...")
	testutil.TestEnv(t)
	testutil.EnsureBuilt(t)

	// Create a VM
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

	// Wait for VM to be fully ready
	t.Log("Waiting for VM networking to be configured...")
	if err := waitForVMReady(t, vmID, 60*time.Second); err != nil {
		t.Fatalf("VM did not become ready: %v", err)
	}

	// Execute a command with flags using -- separator
	t.Log("Running: vers execute", vmID, "--", "ls", "-la")
	out, err = testutil.RunVers(t, testutil.DefaultTimeout, "execute", vmID, "--", "ls", "-la")
	if err != nil {
		t.Fatalf("vers execute with flags failed: %v\nOutput:\n%s", err, out)
	}

	// Verify output contains directory listing elements
	if !strings.Contains(out, "total") || !strings.Contains(out, "root") {
		t.Fatalf("expected directory listing in output, got:\n%s", out)
	}

	t.Log("✓ Execute with flags successful")
	t.Log("TestExecuteWithFlags completed")
}

// TestExecuteQuotedCommand tests execute command with a quoted command string.
func TestExecuteQuotedCommand(t *testing.T) {
	t.Log("Starting TestExecuteQuotedCommand...")
	testutil.TestEnv(t)
	testutil.EnsureBuilt(t)

	// Create a VM
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

	// Wait for VM to be fully ready
	t.Log("Waiting for VM networking to be configured...")
	if err := waitForVMReady(t, vmID, 60*time.Second); err != nil {
		t.Fatalf("VM did not become ready: %v", err)
	}

	// Execute a quoted command
	t.Log("Running: vers execute", vmID, "\"ls -la\"")
	out, err = testutil.RunVers(t, testutil.DefaultTimeout, "execute", vmID, "ls -la")
	if err != nil {
		t.Fatalf("vers execute with quoted command failed: %v\nOutput:\n%s", err, out)
	}

	// Verify output contains directory listing elements
	if !strings.Contains(out, "total") || !strings.Contains(out, "root") {
		t.Fatalf("expected directory listing in output, got:\n%s", out)
	}

	t.Log("✓ Execute with quoted command successful")
	t.Log("TestExecuteQuotedCommand completed")
}

// TestExecuteMultipleArgs tests execute command with multiple separate arguments.
func TestExecuteMultipleArgs(t *testing.T) {
	t.Log("Starting TestExecuteMultipleArgs...")
	testutil.TestEnv(t)
	testutil.EnsureBuilt(t)

	// Create a VM
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

	// Wait for VM to be fully ready
	t.Log("Waiting for VM networking to be configured...")
	if err := waitForVMReady(t, vmID, 60*time.Second); err != nil {
		t.Fatalf("VM did not become ready: %v", err)
	}

	// Execute a command with multiple arguments
	t.Log("Running: vers execute", vmID, "echo", "hello", "world", "from", "vers")
	out, err = testutil.RunVers(t, testutil.DefaultTimeout, "execute", vmID, "echo", "hello", "world", "from", "vers")
	if err != nil {
		t.Fatalf("vers execute with multiple args failed: %v\nOutput:\n%s", err, out)
	}

	// Verify output
	if !strings.Contains(out, "hello world from vers") {
		t.Fatalf("expected 'hello world from vers' in output, got:\n%s", out)
	}

	t.Log("✓ Execute with multiple args successful")
	t.Log("TestExecuteMultipleArgs completed")
}

// TestExecuteInvalidVM tests that execute fails gracefully with a non-existent VM.
func TestExecuteInvalidVM(t *testing.T) {
	t.Log("Starting TestExecuteInvalidVM...")
	testutil.TestEnv(t)
	testutil.EnsureBuilt(t)

	// Try to execute on a non-existent VM
	invalidVMID := "00000000-0000-0000-0000-000000000000"
	t.Logf("Attempting to execute on invalid VM: %s", invalidVMID)

	// We expect this to fail
	out, err := testutil.RunVers(t, testutil.DefaultTimeout, "execute", invalidVMID, "echo", "test")

	if err == nil {
		t.Fatal("expected error when executing on non-existent VM, got nil")
	}

	t.Logf("Got expected error: %v", err)

	// Verify error message mentions VM or connection issue
	if !strings.Contains(strings.ToLower(out), "vm") &&
		!strings.Contains(strings.ToLower(out), "failed") &&
		!strings.Contains(strings.ToLower(out), "not found") {
		t.Logf("Warning: error message could be more specific. Got:\n%s", out)
	}

	t.Log("✓ Execute correctly fails for non-existent VM")
	t.Log("TestExecuteInvalidVM completed")
}

// TestExecuteMissingCommand tests that execute fails with a clear error when no command is provided.
func TestExecuteMissingCommand(t *testing.T) {
	t.Log("Starting TestExecuteMissingCommand...")
	testutil.TestEnv(t)
	testutil.EnsureBuilt(t)

	// Create a VM (we need a valid VM ID for this test)
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

	// Try to execute without a command
	t.Log("Running: vers execute", vmID, "(no command)")
	out, err = testutil.RunVers(t, testutil.DefaultTimeout, "execute", vmID)

	// We expect this to fail
	if err == nil {
		t.Fatal("expected error when executing without a command, got nil")
	}

	t.Logf("Got expected error output:\n%s", out)

	// Verify error message indicates missing arguments
	if !strings.Contains(out, "requires at least 2 arg(s)") {
		t.Fatalf("expected 'requires at least 2 arg(s)' in error, got:\n%s", out)
	}

	// Verify usage is shown
	if !strings.Contains(out, "Usage:") {
		t.Fatalf("expected 'Usage:' in error output, got:\n%s", out)
	}

	t.Log("✓ Execute correctly fails with clear error for missing command")
	t.Log("TestExecuteMissingCommand completed")
}

// TestExecuteCommandWithExitCode tests that execute properly handles non-zero exit codes.
func TestExecuteCommandWithExitCode(t *testing.T) {
	t.Log("Starting TestExecuteCommandWithExitCode...")
	testutil.TestEnv(t)
	testutil.EnsureBuilt(t)

	// Create a VM
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

	// Wait for VM to be fully ready
	t.Log("Waiting for VM networking to be configured...")
	if err := waitForVMReady(t, vmID, 60*time.Second); err != nil {
		t.Fatalf("VM did not become ready: %v", err)
	}

	// Execute a command that exits with non-zero status
	t.Log("Running: vers execute", vmID, "false")
	out, err = testutil.RunVers(t, testutil.DefaultTimeout, "execute", vmID, "false")

	// We expect this to fail
	if err == nil {
		t.Fatal("expected error when command exits with non-zero status, got nil")
	}

	t.Logf("Got expected error for non-zero exit: %v", err)

	// Verify error mentions exit code
	if !strings.Contains(out, "exit") && !strings.Contains(out, "code") {
		t.Logf("Warning: error message could mention exit code. Got:\n%s", out)
	}

	t.Log("✓ Execute properly handles non-zero exit codes")
	t.Log("TestExecuteCommandWithExitCode completed")
}
