package test

import (
	"testing"
)

// TestBranchLifecycle creates a VM, branches it, and cleans up.
func TestBranchLifecycle(t *testing.T) {
	testEnv(t)
	ensureBuilt(t)

	vmAlias := uniqueAlias("smoke")
	branchAlias := vmAlias + "-branch"

	// Start a VM with known alias
	out, err := runVers(t, defaultTimeout, "run", "-N", vmAlias)
	if err != nil {
		t.Fatalf("vers run failed: %v\nOutput:\n%s", err, out)
	}

	// Cleanup VM and all its children at end
	registerVMCleanup(t, vmAlias, true)

	// Branch from the explicitly named root VM alias
	out, err = runVers(t, defaultTimeout, "branch", "-n", branchAlias, vmAlias)
	if err != nil {
		t.Fatalf("vers branch failed: %v\nOutput:\n%s", err, out)
	}
}
