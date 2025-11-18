package test

import (
	"strings"
	"testing"

	"github.com/hdresearch/vers-cli/test/testutil"
)

// TestExecuteRunsCommand creates a VM and executes a simple command on it.
func TestExecuteRunsCommand(t *testing.T) {
	testutil.TestEnv(t)
	testutil.EnsureBuilt(t)

	// Start a VM
	out, err := testutil.RunVers(t, testutil.DefaultTimeout, "run")
	if err != nil {
		t.Fatalf("vers run failed: %v\nOutput:\n%s", err, out)
	}

	// Parse VM ID from output
	vmID, err := testutil.ParseVMID(out)
	if err != nil {
		t.Fatalf("failed to parse VM ID: %v\nOutput:\n%s", err, out)
	}
	testutil.RegisterVMCleanup(t, vmID, false)

	// Execute a basic echo command on the VM
	out, err = testutil.RunVers(t, testutil.DefaultTimeout, "execute", vmID, "echo", "hello-from-vers")
	if err != nil {
		t.Fatalf("vers execute failed: %v\nOutput:\n%s", err, out)
	}
	if !strings.Contains(out, "hello-from-vers") {
		t.Fatalf("expected echoed output from execute, got:\n%s", out)
	}
}

// TestKillNonRecursiveWithChildrenShowsHelpfulMessage ensures kill without -r fails with guidance.
func TestKillNonRecursiveWithChildrenShowsHelpfulMessage(t *testing.T) {
	testutil.TestEnv(t)
	testutil.EnsureBuilt(t)

	// Start a VM
	out, err := testutil.RunVers(t, testutil.DefaultTimeout, "run")
	if err != nil {
		t.Fatalf("vers run failed: %v\nOutput:\n%s", err, out)
	}

	// Parse VM ID from output
	vmID, err := testutil.ParseVMID(out)
	if err != nil {
		t.Fatalf("failed to parse VM ID: %v\nOutput:\n%s", err, out)
	}
	testutil.RegisterVMCleanup(t, vmID, true)

	// Create a child VM (branch A)
	out, err = testutil.RunVers(t, testutil.DefaultTimeout, "branch", vmID)
	if err != nil {
		t.Fatalf("vers branch failed: %v\nOutput:\n%s", err, out)
	}
	branchAID, err := testutil.ParseVMID(out)
	if err != nil {
		t.Fatalf("failed to parse branch A VM ID: %v\nOutput:\n%s", err, out)
	}

	// Create a grandchild VM (branch B from branch A) so branch A has children
	out, err = testutil.RunVers(t, testutil.DefaultTimeout, "branch", branchAID)
	if err != nil {
		t.Fatalf("vers branch (grandchild) failed: %v\nOutput:\n%s", err, out)
	}

	// Attempt to delete the parent VM without -r (skip confirmation)
	out, err = testutil.RunVers(t, testutil.DefaultTimeout, "kill", "-y", branchAID)
	if err == nil {
		t.Fatalf("expected kill to fail without -r for VM with children; output:\n%s", out)
	}
	// Look for friendly guidance about using --recursive
	if !strings.Contains(out, "--recursive (-r)") && !strings.Contains(out, "HasChildren") {
		t.Fatalf("expected friendly guidance for recursive delete, got:\n%s", out)
	}
}
