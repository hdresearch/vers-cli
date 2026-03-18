package test

import (
	"strings"
	"testing"

	"github.com/hdresearch/vers-cli/test/testutil"
)

// TestResizeBasic tests resizing a VM's disk.
func TestResizeBasic(t *testing.T) {
	t.Log("Starting TestResizeBasic...")
	testutil.TestEnv(t)
	testutil.EnsureBuilt(t)

	// Create a VM (default disk size is typically 10240 MiB)
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

	// Resize to a larger size
	t.Logf("Running: vers resize %s --size 20480", vmID)
	out, err = testutil.RunVers(t, testutil.DefaultTimeout, "resize", vmID, "--size", "20480")
	if err != nil {
		t.Fatalf("vers resize failed: %v\nOutput:\n%s", err, out)
	}
	t.Logf("Resize output:\n%s", out)

	if !strings.Contains(out, "resized") {
		t.Fatalf("expected 'resized' in output, got:\n%s", out)
	}
	if !strings.Contains(out, "20480") {
		t.Fatalf("expected '20480' in output, got:\n%s", out)
	}

	t.Log("✓ VM disk resized successfully")
}

// TestResizeTooSmall tests that shrinking a disk fails.
func TestResizeTooSmall(t *testing.T) {
	t.Log("Starting TestResizeTooSmall...")
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
	testutil.RegisterVMCleanup(t, vmID, true)

	// Try to resize to a smaller size (default is 10240, try 1024)
	t.Logf("Running: vers resize %s --size 1024", vmID)
	out, err = testutil.RunVers(t, testutil.DefaultTimeout, "resize", vmID, "--size", "1024")

	if err == nil {
		t.Fatalf("expected error when shrinking disk, got success:\n%s", out)
	}
	t.Logf("Got expected error:\n%s", out)
	t.Log("✓ Resize to smaller size failed as expected")
}

// TestResizeMissingSize tests that --size flag is required.
func TestResizeMissingSize(t *testing.T) {
	t.Log("Starting TestResizeMissingSize...")
	testutil.TestEnv(t)
	testutil.EnsureBuilt(t)

	out, err := testutil.RunVers(t, testutil.DefaultTimeout, "resize", "some-vm")
	if err == nil {
		t.Fatalf("expected error when --size is missing, got success:\n%s", out)
	}

	if !strings.Contains(strings.ToLower(out), "required") && !strings.Contains(strings.ToLower(out), "size") {
		t.Logf("Warning: error message could mention --size requirement. Got:\n%s", out)
	}

	t.Log("✓ Missing --size flag failed as expected")
}

// TestResizeNonExistent tests that resizing a non-existent VM fails.
func TestResizeNonExistent(t *testing.T) {
	t.Log("Starting TestResizeNonExistent...")
	testutil.TestEnv(t)
	testutil.EnsureBuilt(t)

	fakeID := "00000000-0000-0000-0000-000000000000"
	t.Logf("Running: vers resize %s --size 20480", fakeID)
	out, err := testutil.RunVers(t, testutil.DefaultTimeout, "resize", fakeID, "--size", "20480")

	if err == nil {
		t.Fatalf("expected error for non-existent VM, got success:\n%s", out)
	}
	t.Logf("Got expected error:\n%s", out)
	t.Log("✓ Resize non-existent VM failed as expected")
}
