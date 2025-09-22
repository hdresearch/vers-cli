package test

import (
	"strings"
	"testing"
)

// TestRenameVMAndCluster renames both VM and cluster, verifying resolution works.
func TestRenameVMAndCluster(t *testing.T) {
	testEnv(t)
	ensureBuilt(t)

	clusterAlias, vmAlias := uniqueAliases("smoke")
	newVmAlias := vmAlias + "-renamed"
	newClusterAlias := clusterAlias + "-renamed"

	// Start a cluster
	out, err := runVers(t, defaultTimeout, "run", "-n", clusterAlias, "-N", vmAlias)
	if err != nil {
		t.Fatalf("vers run failed: %v\nOutput:\n%s", err, out)
	}
	// Ensure cleanup by original alias
	registerClusterCleanup(t, clusterAlias)

	// Rename VM
	out, err = runVers(t, defaultTimeout, "rename", vmAlias, newVmAlias)
	if err != nil {
		t.Fatalf("vers rename (vm) failed: %v\nOutput:\n%s", err, out)
	}
	// Verify new VM alias resolves via status
	out, err = runVers(t, defaultTimeout, "status", newVmAlias)
	if err != nil {
		t.Fatalf("vers status <new-vm-alias> failed: %v\nOutput:\n%s", err, out)
	}

	// Rename cluster
	out, err = runVers(t, defaultTimeout, "rename", "-c", clusterAlias, newClusterAlias)
	if err != nil {
		t.Fatalf("vers rename (cluster) failed: %v\nOutput:\n%s", err, out)
	}
	// Verify cluster rename via status -c
	out, err = runVers(t, defaultTimeout, "status", "-c", newClusterAlias)
	if err != nil {
		t.Fatalf("vers status -c <new-cluster-alias> failed: %v\nOutput:\n%s", err, out)
	}
	// Add cleanup for new alias too in case original alias no longer resolves
	registerClusterCleanup(t, newClusterAlias)
}

// TestCheckoutUpdatesHead switches HEAD to a VM and verifies the current HEAD display.
func TestCheckoutUpdatesHead(t *testing.T) {
	testEnv(t)
	ensureBuilt(t)

	clusterAlias, vmAlias := uniqueAliases("smoke")

	out, err := runVers(t, defaultTimeout, "run", "-n", clusterAlias, "-N", vmAlias)
	if err != nil {
		t.Fatalf("vers run failed: %v\nOutput:\n%s", err, out)
	}
	registerClusterCleanup(t, clusterAlias)

	// Switch HEAD to the VM by alias
	out, err = runVers(t, defaultTimeout, "checkout", vmAlias)
	if err != nil {
		t.Fatalf("vers checkout <vmAlias> failed: %v\nOutput:\n%s", err, out)
	}

	// Show current HEAD and ensure alias is reflected
	out, err = runVers(t, defaultTimeout, "checkout")
	if err != nil {
		t.Fatalf("vers checkout (show current) failed: %v\nOutput:\n%s", err, out)
	}
	if !strings.Contains(out, vmAlias) || !strings.Contains(out, "Current HEAD:") {
		t.Fatalf("expected current HEAD output to include alias '%s', got:\n%s", vmAlias, out)
	}
}
