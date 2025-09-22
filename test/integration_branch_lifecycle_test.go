package test

import (
	"testing"
)

// TestBranchLifecycle creates a cluster, branches the root VM, and cleans up.
func TestBranchLifecycle(t *testing.T) {
	testEnv(t)
	ensureBuilt(t)

	clusterAlias, vmAlias := uniqueAliases("smoke")
	branchAlias := clusterAlias + "-branch"

	// Start a cluster with known aliases
	out, err := runVers(t, defaultTimeout, "run", "-n", clusterAlias, "-N", vmAlias)
	if err != nil {
		t.Fatalf("vers run failed: %v\nOutput:\n%s", err, out)
	}

	// Cleanup entire cluster at end (will remove branch as well)
	registerClusterCleanup(t, clusterAlias)

	// Branch from the explicitly named root VM alias
	out, err = runVers(t, defaultTimeout, "branch", "-n", branchAlias, vmAlias)
	if err != nil {
		t.Fatalf("vers branch failed: %v\nOutput:\n%s", err, out)
	}
}
