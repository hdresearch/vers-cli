package test

import (
	"strings"
	"testing"

	"github.com/hdresearch/vers-cli/test/testutil"
)

// TestStatusForVMByAlias ensures VM-specific status by alias renders details.
func TestStatusForVMByAlias(t *testing.T) {
	testutil.TestEnv(t)
	testutil.EnsureBuilt(t)

	vmAlias := testutil.UniqueAlias("smoke")
	out, err := testutil.RunVers(t, testutil.DefaultTimeout, "run", "-N", vmAlias)
	if err != nil {
		t.Fatalf("vers run failed: %v\nOutput:\n%s", err, out)
	}
	testutil.RegisterVMCleanup(t, vmAlias, true)

	out, err = testutil.RunVers(t, testutil.DefaultTimeout, "status", vmAlias)
	if err != nil {
		t.Fatalf("vers status <vmAlias> failed: %v\nOutput:\n%s", err, out)
	}
	if !strings.Contains(out, "VM details:") {
		t.Fatalf("expected 'VM details:' in output, got:\n%s", out)
	}
}
