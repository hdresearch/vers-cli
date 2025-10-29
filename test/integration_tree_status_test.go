package test

import (
	"strings"
	"testing"
)

// TestStatusForVMByAlias ensures VM-specific status by alias renders details.
func TestStatusForVMByAlias(t *testing.T) {
	testEnv(t)
	ensureBuilt(t)

	vmAlias := uniqueAlias("smoke")
	out, err := runVers(t, defaultTimeout, "run", "-N", vmAlias)
	if err != nil {
		t.Fatalf("vers run failed: %v\nOutput:\n%s", err, out)
	}
	registerVMCleanup(t, vmAlias, true)

	out, err = runVers(t, defaultTimeout, "status", vmAlias)
	if err != nil {
		t.Fatalf("vers status <vmAlias> failed: %v\nOutput:\n%s", err, out)
	}
	if !strings.Contains(out, "VM details:") {
		t.Fatalf("expected 'VM details:' in output, got:\n%s", out)
	}
}
