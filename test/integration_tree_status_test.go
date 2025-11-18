package test

import (
	"strings"
	"testing"

	"github.com/hdresearch/vers-cli/test/testutil"
)

// TestStatusForVMByAlias ensures VM-specific status by ID renders details.
func TestStatusForVMByAlias(t *testing.T) {
	testutil.TestEnv(t)
	testutil.EnsureBuilt(t)

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

	out, err = testutil.RunVers(t, testutil.DefaultTimeout, "status", vmID)
	if err != nil {
		t.Fatalf("vers status <vmID> failed: %v\nOutput:\n%s", err, out)
	}
	if !strings.Contains(out, "VM details:") {
		t.Fatalf("expected 'VM details:' in output, got:\n%s", out)
	}
}
