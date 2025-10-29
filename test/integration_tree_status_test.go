package test

import (
	"strings"
	"testing"
)

// TestTreeDisplaysClusterStructureAndHead builds a small tree and validates output markers.
func TestTreeDisplaysClusterStructureAndHead(t *testing.T) {
	t.Skip("Tree view deprecated - cluster concept removed from SDK")
	testEnv(t)
	ensureBuilt(t)

	clusterAlias, vmAlias := uniqueAliases("smoke")
	childA := vmAlias + "-a"
	childB := vmAlias + "-b"

	// Start cluster and create two children under root
	out, err := runVers(t, defaultTimeout, "run", "-n", clusterAlias, "-N", vmAlias)
	if err != nil {
		t.Fatalf("vers run failed: %v\nOutput:\n%s", err, out)
	}
	registerClusterCleanup(t, clusterAlias)

	// Branch children from root
	if out, err = runVers(t, defaultTimeout, "branch", "-n", childA, vmAlias); err != nil {
		t.Fatalf("branch A failed: %v\nOutput:\n%s", err, out)
	}
	if out, err = runVers(t, defaultTimeout, "branch", "-n", childB, vmAlias); err != nil {
		t.Fatalf("branch B failed: %v\nOutput:\n%s", err, out)
	}

	// Mark HEAD on childA to check highlighting
	if out, err = runVers(t, defaultTimeout, "checkout", childA); err != nil {
		t.Fatalf("checkout failed: %v\nOutput:\n%s", err, out)
	}

	// Render tree for the cluster by alias
	out, err = runVers(t, defaultTimeout, "tree", clusterAlias)
	if err != nil {
		t.Fatalf("vers tree <clusterAlias> failed: %v\nOutput:\n%s", err, out)
	}

	// Validate key markers exist
	if !strings.Contains(out, "Cluster:") || !strings.Contains(out, "Total VMs:") {
		t.Fatalf("expected cluster header in tree output, got:\n%s", out)
	}
	if !strings.Contains(out, "├── ") && !strings.Contains(out, "└── ") {
		t.Fatalf("expected tree connectors in output, got:\n%s", out)
	}
	if !strings.Contains(out, childA) || !strings.Contains(out, "<- HEAD") {
		t.Fatalf("expected HEAD marker on childA in tree output, got:\n%s", out)
	}
	// Legend present
	if !strings.Contains(out, "Legend:") || !strings.Contains(out, "[R]") {
		t.Fatalf("expected legend in tree output, got:\n%s", out)
	}
}

// TestStatusForVMByAlias ensures VM-specific status by alias renders details.
func TestStatusForVMByAlias(t *testing.T) {
	testEnv(t)
	ensureBuilt(t)

	clusterAlias, vmAlias := uniqueAliases("smoke")
	out, err := runVers(t, defaultTimeout, "run", "-n", clusterAlias, "-N", vmAlias)
	if err != nil {
		t.Fatalf("vers run failed: %v\nOutput:\n%s", err, out)
	}
	registerClusterCleanup(t, clusterAlias)

	out, err = runVers(t, defaultTimeout, "status", vmAlias)
	if err != nil {
		t.Fatalf("vers status <vmAlias> failed: %v\nOutput:\n%s", err, out)
	}
	if !strings.Contains(out, "VM details:") {
		t.Fatalf("expected 'VM details:' in output, got:\n%s", out)
	}
}

// TestTreeUsesHeadCluster verifies that `vers tree` with no args uses the cluster containing HEAD.
func TestTreeUsesHeadCluster(t *testing.T) {
	t.Skip("Tree view deprecated - cluster concept removed from SDK")
	testEnv(t)
	ensureBuilt(t)

	clusterAlias, vmAlias := uniqueAliases("smoke")
	out, err := runVers(t, defaultTimeout, "run", "-n", clusterAlias, "-N", vmAlias)
	if err != nil {
		t.Fatalf("vers run failed: %v\nOutput:\n%s", err, out)
	}
	registerClusterCleanup(t, clusterAlias)

	// Set HEAD to the root VM by alias
	if out, err = runVers(t, defaultTimeout, "checkout", vmAlias); err != nil {
		t.Fatalf("vers checkout failed: %v\nOutput:\n%s", err, out)
	}

	// Call tree with no args; it should resolve cluster via HEAD
	out, err = runVers(t, defaultTimeout, "tree")
	if err != nil {
		t.Fatalf("vers tree (no args) failed: %v\nOutput:\n%s", err, out)
	}
	if !strings.Contains(out, "Cluster:") || !strings.Contains(out, vmAlias) || !strings.Contains(out, "<- HEAD") {
		t.Fatalf("expected tree output to contain cluster header and HEAD VM alias, got:\n%s", out)
	}
}
