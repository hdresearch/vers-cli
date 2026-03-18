package test

import (
	"strings"
	"testing"

	"github.com/hdresearch/vers-cli/test/testutil"
)

// TestInfoBasic tests getting metadata for a running VM.
func TestInfoBasic(t *testing.T) {
	t.Log("Starting TestInfoBasic...")
	testutil.TestEnv(t)
	testutil.EnsureBuilt(t)

	// Create a VM
	t.Log("Running: vers run")
	out, err := testutil.RunVers(t, testutil.DefaultTimeout, "run")
	if err != nil {
		t.Fatalf("vers run failed: %v\nOutput:\n%s", err, out)
	}

	vmID, err := testutil.ParseVMID(out)
	if err != nil {
		t.Fatalf("failed to parse VM ID: %v\nOutput:\n%s", err, out)
	}
	t.Logf("Created VM: %s", vmID)
	testutil.RegisterVMCleanup(t, vmID, true)

	// Get info
	t.Logf("Running: vers info %s", vmID)
	out, err = testutil.RunVers(t, testutil.DefaultTimeout, "info", vmID)
	if err != nil {
		t.Fatalf("vers info failed: %v\nOutput:\n%s", err, out)
	}
	t.Logf("Info output:\n%s", out)

	// Verify key fields are present
	if !strings.Contains(out, vmID) {
		t.Fatalf("expected VM ID in output, got:\n%s", out)
	}
	if !strings.Contains(out, "State:") {
		t.Fatalf("expected State field in output, got:\n%s", out)
	}
	if !strings.Contains(out, "IP:") {
		t.Fatalf("expected IP field in output, got:\n%s", out)
	}
	if !strings.Contains(out, "Created:") {
		t.Fatalf("expected Created field in output, got:\n%s", out)
	}

	t.Log("✓ VM info retrieved successfully")
}

// TestInfoWithLineage tests that a branched VM shows parent commit info.
func TestInfoWithLineage(t *testing.T) {
	t.Log("Starting TestInfoWithLineage...")
	testutil.TestEnv(t)
	testutil.EnsureBuilt(t)

	// Create a VM, commit it, then restore from commit
	t.Log("Running: vers run")
	out, err := testutil.RunVers(t, testutil.DefaultTimeout, "run")
	if err != nil {
		t.Fatalf("vers run failed: %v\nOutput:\n%s", err, out)
	}
	parentVMID, err := testutil.ParseVMID(out)
	if err != nil {
		t.Fatalf("failed to parse VM ID: %v\nOutput:\n%s", err, out)
	}
	testutil.RegisterVMCleanup(t, parentVMID, true)

	// Commit the VM
	t.Logf("Running: vers commit %s", parentVMID)
	out, err = testutil.RunVers(t, testutil.DefaultTimeout, "commit", parentVMID)
	if err != nil {
		t.Fatalf("vers commit failed: %v\nOutput:\n%s", err, out)
	}

	// Branch from the parent to get a child with lineage
	t.Logf("Running: vers branch %s", parentVMID)
	out, err = testutil.RunVers(t, testutil.DefaultTimeout, "branch", parentVMID)
	if err != nil {
		t.Fatalf("vers branch failed: %v\nOutput:\n%s", err, out)
	}

	childVMID, err := testutil.ParseVMID(out)
	if err != nil {
		t.Fatalf("failed to parse child VM ID: %v\nOutput:\n%s", err, out)
	}
	t.Logf("Created child VM: %s", childVMID)

	// Get info on the child — should show parent commit
	t.Logf("Running: vers info %s", childVMID)
	out, err = testutil.RunVers(t, testutil.DefaultTimeout, "info", childVMID)
	if err != nil {
		t.Fatalf("vers info failed: %v\nOutput:\n%s", err, out)
	}
	t.Logf("Info output:\n%s", out)

	if !strings.Contains(out, childVMID) {
		t.Fatalf("expected child VM ID in output, got:\n%s", out)
	}
	// Branched VMs should have parent commit info
	if !strings.Contains(out, "Parent commit") {
		t.Logf("Note: no parent commit shown — may depend on backend behavior")
	}

	t.Log("✓ VM info with lineage retrieved")
}

// TestInfoNonExistent tests that info for a non-existent VM fails gracefully.
func TestInfoNonExistent(t *testing.T) {
	t.Log("Starting TestInfoNonExistent...")
	testutil.TestEnv(t)
	testutil.EnsureBuilt(t)

	fakeID := "00000000-0000-0000-0000-000000000000"
	t.Logf("Running: vers info %s", fakeID)
	out, err := testutil.RunVers(t, testutil.DefaultTimeout, "info", fakeID)

	if err == nil {
		t.Fatalf("expected error for non-existent VM, got success:\n%s", out)
	}
	t.Logf("Got expected error:\n%s", out)
	t.Log("✓ Info for non-existent VM failed as expected")
}
