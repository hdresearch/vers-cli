package test

import (
	"regexp"
	"strings"
	"testing"

	"github.com/hdresearch/vers-cli/test/testutil"
)

// TestCommitAndList tests committing a VM and then listing commits.
func TestCommitAndList(t *testing.T) {
	t.Log("Starting TestCommitAndList...")
	testutil.TestEnv(t)
	testutil.EnsureBuilt(t)

	// Create a VM to commit
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

	// Commit the VM
	t.Logf("Running: vers commit %s", vmID)
	out, err = testutil.RunVers(t, testutil.DefaultTimeout, "commit", vmID)
	if err != nil {
		t.Fatalf("vers commit failed: %v\nOutput:\n%s", err, out)
	}
	t.Logf("Commit output:\n%s", out)

	if !strings.Contains(out, "Successfully committed") {
		t.Fatalf("expected success message, got:\n%s", out)
	}

	// Extract commit ID
	re := regexp.MustCompile(`Commit ID:\s*([0-9a-f-]+)`)
	matches := re.FindStringSubmatch(out)
	if len(matches) < 2 {
		t.Fatalf("failed to extract commit ID from output:\n%s", out)
	}
	commitID := matches[1]
	t.Logf("Created commit: %s", commitID)

	// List commits and verify ours shows up
	t.Log("Running: vers commit list")
	out, err = testutil.RunVers(t, testutil.DefaultTimeout, "commit", "list")
	if err != nil {
		t.Fatalf("vers commit list failed: %v\nOutput:\n%s", err, out)
	}
	t.Logf("Commit list output:\n%s", out)

	if !strings.Contains(out, commitID) {
		t.Fatalf("expected commit %s in list output, got:\n%s", commitID, out)
	}

	t.Log("✓ Commit created and listed successfully")
}

// TestCommitHistory tests the commit history (parents) command.
func TestCommitHistory(t *testing.T) {
	t.Log("Starting TestCommitHistory...")
	testutil.TestEnv(t)
	testutil.EnsureBuilt(t)

	// Create a VM and commit it
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

	t.Logf("Running: vers commit %s", vmID)
	out, err = testutil.RunVers(t, testutil.DefaultTimeout, "commit", vmID)
	if err != nil {
		t.Fatalf("vers commit failed: %v\nOutput:\n%s", err, out)
	}

	re := regexp.MustCompile(`Commit ID:\s*([0-9a-f-]+)`)
	matches := re.FindStringSubmatch(out)
	if len(matches) < 2 {
		t.Fatalf("failed to extract commit ID from output:\n%s", out)
	}
	commitID := matches[1]

	// Get commit history
	t.Logf("Running: vers commit history %s", commitID)
	out, err = testutil.RunVers(t, testutil.DefaultTimeout, "commit", "history", commitID)
	if err != nil {
		t.Fatalf("vers commit history failed: %v\nOutput:\n%s", err, out)
	}
	t.Logf("History output:\n%s", out)

	if !strings.Contains(out, "Commit History") {
		t.Fatalf("expected 'Commit History' header in output, got:\n%s", out)
	}

	t.Log("✓ Commit history retrieved successfully")
}

// TestCommitPublishUnpublish tests publishing and unpublishing a commit.
func TestCommitPublishUnpublish(t *testing.T) {
	t.Log("Starting TestCommitPublishUnpublish...")
	testutil.TestEnv(t)
	testutil.EnsureBuilt(t)

	// Create and commit a VM
	out, err := testutil.RunVers(t, testutil.DefaultTimeout, "run")
	if err != nil {
		t.Fatalf("vers run failed: %v\nOutput:\n%s", err, out)
	}
	vmID, err := testutil.ParseVMID(out)
	if err != nil {
		t.Fatalf("failed to parse VM ID: %v\nOutput:\n%s", err, out)
	}
	testutil.RegisterVMCleanup(t, vmID, true)

	out, err = testutil.RunVers(t, testutil.DefaultTimeout, "commit", vmID)
	if err != nil {
		t.Fatalf("vers commit failed: %v\nOutput:\n%s", err, out)
	}

	re := regexp.MustCompile(`Commit ID:\s*([0-9a-f-]+)`)
	matches := re.FindStringSubmatch(out)
	if len(matches) < 2 {
		t.Fatalf("failed to extract commit ID:\n%s", out)
	}
	commitID := matches[1]

	// Publish
	t.Logf("Running: vers commit publish %s", commitID)
	out, err = testutil.RunVers(t, testutil.DefaultTimeout, "commit", "publish", commitID)
	if err != nil {
		t.Fatalf("vers commit publish failed: %v\nOutput:\n%s", err, out)
	}
	if !strings.Contains(out, "now public") {
		t.Fatalf("expected 'now public' in output, got:\n%s", out)
	}

	// Unpublish
	t.Logf("Running: vers commit unpublish %s", commitID)
	out, err = testutil.RunVers(t, testutil.DefaultTimeout, "commit", "unpublish", commitID)
	if err != nil {
		t.Fatalf("vers commit unpublish failed: %v\nOutput:\n%s", err, out)
	}
	if !strings.Contains(out, "now private") {
		t.Fatalf("expected 'now private' in output, got:\n%s", out)
	}

	t.Log("✓ Commit publish/unpublish works")
}

// TestCommitDeleteNonExistent tests that deleting a non-existent commit fails gracefully.
func TestCommitDeleteNonExistent(t *testing.T) {
	t.Log("Starting TestCommitDeleteNonExistent...")
	testutil.TestEnv(t)
	testutil.EnsureBuilt(t)

	fakeID := "00000000-0000-0000-0000-000000000000"
	t.Logf("Running: vers commit delete %s", fakeID)
	out, err := testutil.RunVers(t, testutil.DefaultTimeout, "commit", "delete", fakeID)

	if err == nil {
		t.Fatalf("expected error deleting non-existent commit, got success:\n%s", out)
	}
	t.Logf("Got expected error:\n%s", out)
	t.Log("✓ Delete non-existent commit failed as expected")
}

// TestCommitListEmpty tests that commit list works even with no commits.
func TestCommitListEmpty(t *testing.T) {
	t.Log("Starting TestCommitListEmpty...")
	testutil.TestEnv(t)
	testutil.EnsureBuilt(t)

	// Just verify the command doesn't error out
	t.Log("Running: vers commit list")
	out, err := testutil.RunVers(t, testutil.DefaultTimeout, "commit", "list")
	if err != nil {
		t.Fatalf("vers commit list failed: %v\nOutput:\n%s", err, out)
	}

	// Should show either commits or "No commits found"
	if !strings.Contains(out, "Commits") && !strings.Contains(out, "commit") {
		t.Fatalf("expected commit-related output, got:\n%s", out)
	}

	t.Log("✓ Commit list works")
}
