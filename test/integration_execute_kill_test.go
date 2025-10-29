package test

import (
	"strings"
	"testing"
)

// TestExecuteRunsCommand creates a VM and executes a simple command on it.
func TestExecuteRunsCommand(t *testing.T) {
	testEnv(t)
	ensureBuilt(t)

	vmAlias := uniqueAlias("smoke")

	// Start a VM
	out, err := runVers(t, defaultTimeout, "run", "-N", vmAlias)
	if err != nil {
		t.Fatalf("vers run failed: %v\nOutput:\n%s", err, out)
	}
	registerVMCleanup(t, vmAlias, true)

	// Execute a basic echo command on the VM
	out, err = runVers(t, defaultTimeout, "execute", vmAlias, "echo", "hello-from-vers")
	if err != nil {
		t.Fatalf("vers execute failed: %v\nOutput:\n%s", err, out)
	}
	if !strings.Contains(out, "hello-from-vers") {
		t.Fatalf("expected echoed output from execute, got:\n%s", out)
	}
}

// TestKillNonRecursiveWithChildrenShowsHelpfulMessage ensures kill without -r fails with guidance.
func TestKillNonRecursiveWithChildrenShowsHelpfulMessage(t *testing.T) {
	testEnv(t)
	ensureBuilt(t)

	vmAlias := uniqueAlias("smoke")

	// Start a VM
	out, err := runVers(t, defaultTimeout, "run", "-N", vmAlias)
	if err != nil {
		t.Fatalf("vers run failed: %v\nOutput:\n%s", err, out)
	}
	registerVMCleanup(t, vmAlias, true)

	// Create a child VM (branch A)
	branchA := vmAlias + "-a"
	out, err = runVers(t, defaultTimeout, "branch", "-n", branchA, vmAlias)
	if err != nil {
		t.Fatalf("vers branch failed: %v\nOutput:\n%s", err, out)
	}
	// Create a grandchild VM (branch B from branch A) so branch A has children
	branchB := vmAlias + "-b"
	out, err = runVers(t, defaultTimeout, "branch", "-n", branchB, branchA)
	if err != nil {
		t.Fatalf("vers branch (grandchild) failed: %v\nOutput:\n%s", err, out)
	}

	// Attempt to delete the parent VM without -r (skip confirmation)
	out, err = runVers(t, defaultTimeout, "kill", "-y", branchA)
	if err == nil {
		t.Fatalf("expected kill to fail without -r for VM with children; output:\n%s", out)
	}
	// Look for friendly guidance about using --recursive
	if !strings.Contains(out, "--recursive (-r)") && !strings.Contains(out, "HasChildren") {
		t.Fatalf("expected friendly guidance for recursive delete, got:\n%s", out)
	}
}
