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

	vmAlias := testutil.UniqueAlias("smoke")

	// Start a VM
	out, err := testutil.RunVers(t, testutil.DefaultTimeout, "run", "-N", vmAlias)
	if err != nil {
		t.Fatalf("vers run failed: %v\nOutput:\n%s", err, out)
	}
	testutil.RegisterVMCleanup(t, vmAlias, true)

	// Execute a basic echo command on the VM
	out, err = testutil.RunVers(t, testutil.DefaultTimeout, "execute", vmAlias, "echo", "hello-from-vers")
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

	vmAlias := testutil.UniqueAlias("smoke")

	// Start a VM
	out, err := testutil.RunVers(t, testutil.DefaultTimeout, "run", "-N", vmAlias)
	if err != nil {
		t.Fatalf("vers run failed: %v\nOutput:\n%s", err, out)
	}
	testutil.RegisterVMCleanup(t, vmAlias, true)

	// Create a child VM (branch A)
	branchA := vmAlias + "-a"
	out, err = testutil.RunVers(t, testutil.DefaultTimeout, "branch", "-n", branchA, vmAlias)
	if err != nil {
		t.Fatalf("vers branch failed: %v\nOutput:\n%s", err, out)
	}
	// Create a grandchild VM (branch B from branch A) so branch A has children
	branchB := vmAlias + "-b"
	out, err = testutil.RunVers(t, testutil.DefaultTimeout, "branch", "-n", branchB, branchA)
	if err != nil {
		t.Fatalf("vers branch (grandchild) failed: %v\nOutput:\n%s", err, out)
	}

	// Attempt to delete the parent VM without -r (skip confirmation)
	out, err = testutil.RunVers(t, testutil.DefaultTimeout, "kill", "-y", branchA)
	if err == nil {
		t.Fatalf("expected kill to fail without -r for VM with children; output:\n%s", out)
	}
	// Look for friendly guidance about using --recursive
	if !strings.Contains(out, "--recursive (-r)") && !strings.Contains(out, "HasChildren") {
		t.Fatalf("expected friendly guidance for recursive delete, got:\n%s", out)
	}
}
