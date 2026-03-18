package test

import (
	"strings"
	"testing"
	"time"

	"github.com/hdresearch/vers-cli/test/testutil"
)

// TestExecuteRunsCommand creates a VM and executes a simple command on it.
func TestExecuteRunsCommand(t *testing.T) {
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
	testutil.RegisterVMCleanup(t, vmID)

	// Wait for VM networking to be configured (WireGuard, DNAT rules)
	t.Logf("Waiting for VM networking to be ready...")
	time.Sleep(15 * time.Second)

	// Execute a basic echo command on the VM
	out, err = testutil.RunVers(t, testutil.DefaultTimeout, "execute", vmID, "echo", "hello-from-vers")
	if err != nil {
		t.Fatalf("vers execute failed: %v\nOutput:\n%s", err, out)
	}
	if !strings.Contains(out, "hello-from-vers") {
		t.Fatalf("expected echoed output from execute, got:\n%s", out)
	}
}
