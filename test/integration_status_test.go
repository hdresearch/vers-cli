package test

import (
	"strings"
	"testing"
	"time"

	"github.com/hdresearch/vers-cli/test/testutil"
)

// TestStatus_Smoke verifies we can reach the backend and list status.
func TestStatus_Smoke(t *testing.T) {
	testutil.TestEnv(t)
	testutil.EnsureBuilt(t)

	out, err := testutil.RunVers(t, 30*time.Second, "status")
	if err != nil {
		t.Fatalf("vers status failed: %v\nOutput:\n%s", err, out)
	}

	// Basic sanity: output includes common markers indicating successful execution.
	ok := strings.Contains(out, "VM details:") ||
		strings.Contains(out, "Tip:") ||
		strings.Contains(out, "No VMs found.") ||
		strings.Contains(out, "Fetching list of VMs")
	if !ok {
		t.Fatalf("unexpected status output; got:\n%s", out)
	}
}
