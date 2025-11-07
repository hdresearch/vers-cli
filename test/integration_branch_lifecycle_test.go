package test

import (
	"testing"

	"github.com/hdresearch/vers-cli/test/testutil"
)

// TestBranchLifecycle creates a VM, branches it, and cleans up.
func TestBranchLifecycle(t *testing.T) {
	testutil.TestEnv(t)
	testutil.EnsureBuilt(t)

	vmAlias := testutil.UniqueAlias("smoke")
	branchAlias := vmAlias + "-branch"

	// Start a VM with known alias
	out, err := testutil.RunVers(t, testutil.DefaultTimeout, "run", "-N", vmAlias)
	if err != nil {
		t.Fatalf("vers run failed: %v\nOutput:\n%s", err, out)
	}

	// Cleanup VM and all its children at end
	testutil.RegisterVMCleanup(t, vmAlias, true)

	// Branch from the explicitly named root VM alias
	out, err = testutil.RunVers(t, testutil.DefaultTimeout, "branch", "-n", branchAlias, vmAlias)
	if err != nil {
		t.Fatalf("vers branch failed: %v\nOutput:\n%s", err, out)
	}
}
