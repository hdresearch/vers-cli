package test

import (
	"testing"

	"github.com/hdresearch/vers-cli/test/testutil"
)

// TestBranchLifecycle creates a VM, branches it, and cleans up.
func TestBranchLifecycle(t *testing.T) {
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

	// Cleanup VM and all its children at end
	testutil.RegisterVMCleanup(t, vmID, true)

	// Branch from the root VM
	out, err = testutil.RunVers(t, testutil.DefaultTimeout, "branch", vmID)
	if err != nil {
		t.Fatalf("vers branch failed: %v\nOutput:\n%s", err, out)
	}
}
