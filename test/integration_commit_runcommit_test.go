package test

import (
	"regexp"
	"testing"

	"github.com/hdresearch/vers-cli/test/testutil"
)

// TestCommitAndRunCommit commits a VM and starts a new VM from that commit.
func TestCommitAndRunCommit(t *testing.T) {
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
	testutil.RegisterVMCleanup(t, vmID, true)

	// Commit the VM; capture Commit ID from output
	out, err = testutil.RunVers(t, testutil.DefaultTimeout, "commit", vmID)
	if err != nil {
		if regexp.MustCompile(`(?i)Error uploading commit to S3|AWS CLI|S3 bucket`).FindString(out) != "" {
			t.Skipf("skipping commit test due to backend storage configuration: %v\nOutput:\n%s", err, out)
			return
		}
		if regexp.MustCompile(`(?i)500 Internal Server Error|Internal server error`).FindString(out) != "" {
			t.Skipf("skipping commit test due to backend server error: %v\nOutput:\n%s", err, out)
			return
		}
		t.Fatalf("vers commit failed: %v\nOutput:\n%s", err, out)
	}
	re := regexp.MustCompile(`(?m)^Commit ID:\s*([\w-]+)\s*$`)
	m := re.FindStringSubmatch(out)
	if len(m) != 2 {
		t.Fatalf("failed to extract commit ID from output:\n%s", out)
	}
	commitID := m[1]

	// Start a new VM from the commit
	out, err = testutil.RunVers(t, testutil.DefaultTimeout, "run-commit", commitID)
	if err != nil {
		t.Fatalf("vers run-commit failed: %v\nOutput:\n%s", err, out)
	}

	// Parse new VM ID from output
	newVmID, err := testutil.ParseVMID(out)
	if err != nil {
		t.Fatalf("failed to parse new VM ID: %v\nOutput:\n%s", err, out)
	}
	testutil.RegisterVMCleanup(t, newVmID, true)

	// Verify status resolves for the new VM
	out, err = testutil.RunVers(t, testutil.DefaultTimeout, "status", newVmID)
	if err != nil {
		t.Fatalf("vers status <new-from-commit> failed: %v\nOutput:\n%s", err, out)
	}
}
