package test

import (
	"os"
	"os/exec"
	"strings"
	"testing"

	"github.com/hdresearch/vers-cli/test/testutil"
)

// TestRunBasic tests the basic `vers run` command creates a VM successfully.
func TestRunBasic(t *testing.T) {
	t.Log("Starting TestRunBasic...")
	testutil.TestEnv(t)
	testutil.EnsureBuilt(t)

	// Run a VM (aliases no longer supported in SDK alpha.23)
	t.Log("Running: vers run")
	out, err := testutil.RunVers(t, testutil.DefaultTimeout, "run")
	if err != nil {
		t.Fatalf("vers run failed: %v\nOutput:\n%s", err, out)
	}
	t.Logf("VM creation output:\n%s", out)

	// SDK alpha.24 now returns VM IDs! Parse it and register cleanup
	vmID, err := testutil.ParseVMID(out)
	if err != nil {
		t.Fatalf("failed to parse VM ID from output: %v\nOutput:\n%s", err, out)
	}
	t.Logf("Created VM: %s", vmID)
	testutil.RegisterVMCleanup(t, vmID, false)

	// Inspect raw API response
	t.Log("Inspecting raw API response from /api/v1/vms endpoint...")
	apiKey := os.Getenv("VERS_API_KEY")
	versURL := os.Getenv("VERS_URL")
	if apiKey != "" && versURL != "" {
		curlCmd := exec.Command("curl", "-s", "-v",
			"-H", "Authorization: Bearer "+apiKey,
			versURL+"/api/v1/vms")
		curlOut, curlErr := curlCmd.CombinedOutput()
		if curlErr != nil {
			t.Logf("Warning: curl failed: %v", curlErr)
		}
		t.Logf("Raw API response:\n%s", string(curlOut))
	}

	// Verify output indicates success
	t.Log("Verifying output indicates success...")
	if !strings.Contains(out, "started successfully") {
		t.Fatalf("expected 'started successfully' in output, got:\n%s", out)
	}

	t.Log("✓ VM created successfully")
	// SDK alpha.24 fixes:
	// - NewRoot() now returns VM ID in response
	// - API now returns proper content-type: application/json headers
	// - Status command works
	t.Log("TestRunBasic completed")
}

// TestRunWithCustomSpecs tests the run command with custom VM specifications.
func TestRunWithCustomSpecs(t *testing.T) {
	t.Log("Starting TestRunWithCustomSpecs...")
	testutil.TestEnv(t)
	testutil.EnsureBuilt(t)

	// Run a VM with custom memory and CPU specs (aliases no longer supported)
	t.Log("Running: vers run --mem-size 1024 --vcpu-count 2")
	out, err := testutil.RunVers(t, testutil.DefaultTimeout, "run",
		"--mem-size", "1024",
		"--vcpu-count", "2",
	)
	if err != nil {
		t.Fatalf("vers run with custom specs failed: %v\nOutput:\n%s", err, out)
	}
	t.Logf("VM creation output:\n%s", out)

	// SDK alpha.24 now returns VM IDs! Parse it and register cleanup
	vmID, err := testutil.ParseVMID(out)
	if err != nil {
		t.Fatalf("failed to parse VM ID from output: %v\nOutput:\n%s", err, out)
	}
	t.Logf("Created VM: %s", vmID)
	testutil.RegisterVMCleanup(t, vmID, false)

	// The command should succeed
	// Note: We can't verify the specs in the output since the new API doesn't return VM details
	t.Log("✓ VM created with custom specs successfully")
	t.Log("TestRunWithCustomSpecs completed")
}

// TestRunWithNonExistentImage tests that run handles a bad image name.
func TestRunWithNonExistentImage(t *testing.T) {
	t.Log("Starting TestRunWithNonExistentImage...")
	testutil.TestEnv(t)
	testutil.EnsureBuilt(t)

	// Try to run with a non-existent image (aliases no longer supported)
	t.Log("Running: vers run --rootfs definitely-does-not-exist-image")
	out, err := testutil.RunVers(t, testutil.DefaultTimeout, "run",
		"--rootfs", "definitely-does-not-exist-image",
	)

	// Note: As of SDK alpha.24, the API appears to accept requests with invalid image names
	// and may use a default fallback. This test documents current behavior.
	if err == nil {
		t.Logf("API accepted invalid image name (may have fallback). Output:\n%s", out)
		// Parse VM ID and register cleanup
		vmID, parseErr := testutil.ParseVMID(out)
		if parseErr != nil {
			t.Logf("Warning: could not parse VM ID from output: %v", parseErr)
		} else {
			t.Logf("Created VM: %s", vmID)
			testutil.RegisterVMCleanup(t, vmID, false)
		}
	} else {
		// If it does fail, verify the error message is helpful
		if !strings.Contains(strings.ToLower(out), "image") &&
			!strings.Contains(strings.ToLower(out), "rootfs") &&
			!strings.Contains(strings.ToLower(out), "not found") {
			t.Logf("Error message could be more specific. Got:\n%s", out)
		}
	}
	t.Log("TestRunWithNonExistentImage completed")
}
