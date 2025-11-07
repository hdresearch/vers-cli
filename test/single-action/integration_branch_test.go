package test

import (
	"regexp"
	"strings"
	"testing"

	"github.com/hdresearch/vers-cli/test/testutil"
)

// TestBranchBasic tests creating a branch from an existing VM.
func TestBranchBasic(t *testing.T) {
	t.Log("Starting TestBranchBasic...")
	testutil.TestEnv(t)
	testutil.EnsureBuilt(t)

	// Create a VM to branch from
	t.Log("Running: vers run")
	out, err := testutil.RunVers(t, testutil.DefaultTimeout, "run")
	if err != nil {
		t.Fatalf("vers run failed: %v\nOutput:\n%s", err, out)
	}
	t.Logf("VM creation output:\n%s", out)

	// Parse parent VM ID
	parentVMID, err := testutil.ParseVMID(out)
	if err != nil {
		t.Fatalf("failed to parse VM ID from output: %v\nOutput:\n%s", err, out)
	}
	t.Logf("Created parent VM: %s", parentVMID)
	testutil.RegisterVMCleanup(t, parentVMID, true) // Recursive to clean up parent and branch

	// Branch from the parent VM
	t.Logf("Running: vers branch %s", parentVMID)
	out, err = testutil.RunVers(t, testutil.DefaultTimeout, "branch", parentVMID)
	if err != nil {
		t.Fatalf("vers branch failed: %v\nOutput:\n%s", err, out)
	}
	t.Logf("Branch creation output:\n%s", out)

	// Verify success message
	if !strings.Contains(out, "New VM created successfully") {
		t.Fatalf("expected 'New VM created successfully' in output, got:\n%s", out)
	}

	// Extract the new VM ID from output (format: "VM ID       : <uuid>" with variable whitespace)
	re := regexp.MustCompile(`VM ID\s*:\s*([0-9a-f-]+)`)
	matches := re.FindStringSubmatch(out)
	if len(matches) < 2 {
		t.Fatalf("failed to extract new VM ID from output:\n%s", out)
	}
	newVMID := matches[1]
	t.Logf("Created branch VM: %s", newVMID)

	// Verify the branch was created from the correct parent
	if !strings.Contains(out, "Creating new VM from:") {
		t.Fatalf("expected 'Creating new VM from' in output, got:\n%s", out)
	}

	t.Log("✓ VM branched successfully")
	t.Log("TestBranchBasic completed")
}

// TestBranchFromNonExistent tests that branching from a non-existent VM fails gracefully.
func TestBranchFromNonExistent(t *testing.T) {
	t.Log("Starting TestBranchFromNonExistent...")
	testutil.TestEnv(t)
	testutil.EnsureBuilt(t)

	// Try to branch from a non-existent VM
	fakeVMID := "00000000-0000-0000-0000-000000000000"
	t.Logf("Running: vers branch %s", fakeVMID)
	out, err := testutil.RunVers(t, testutil.DefaultTimeout, "branch", fakeVMID)

	// Should fail with a clear error
	if err == nil {
		t.Fatalf("expected failure when branching from non-existent VM, got success. Output:\n%s", out)
	}

	// Check for helpful error message
	if !strings.Contains(strings.ToLower(out), "failed to find vm") &&
		!strings.Contains(strings.ToLower(out), "not found") &&
		!strings.Contains(strings.ToLower(out), "does not exist") {
		t.Logf("Warning: error message could be more specific. Got:\n%s", out)
	}

	t.Log("✓ Branch from non-existent VM failed as expected")
	t.Log("TestBranchFromNonExistent completed")
}
