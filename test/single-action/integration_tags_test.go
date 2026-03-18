package test

import (
	"regexp"
	"strings"
	"testing"

	"github.com/hdresearch/vers-cli/test/testutil"
)

// TestTagLifecycle tests the full tag CRUD lifecycle:
// create a VM → commit → tag → list → get → update → delete.
func TestTagLifecycle(t *testing.T) {
	t.Log("Starting TestTagLifecycle...")
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
		t.Fatalf("failed to extract commit ID:\n%s", out)
	}
	commitID := matches[1]
	t.Logf("Created commit: %s", commitID)

	// Use a unique tag name to avoid collisions
	tagName := testutil.UniqueAlias("test-tag")

	// Clean up the tag at test end in case something fails
	t.Cleanup(func() {
		testutil.RunVers(t, testutil.DefaultTimeout, "tag", "delete", tagName)
	})

	// === CREATE ===
	t.Logf("Running: vers tag create %s %s -d 'integration test'", tagName, commitID)
	out, err = testutil.RunVers(t, testutil.DefaultTimeout, "tag", "create", tagName, commitID, "-d", "integration test")
	if err != nil {
		t.Fatalf("vers tag create failed: %v\nOutput:\n%s", err, out)
	}
	if !strings.Contains(out, tagName) {
		t.Fatalf("expected tag name in output, got:\n%s", out)
	}
	if !strings.Contains(out, commitID) {
		t.Fatalf("expected commit ID in output, got:\n%s", out)
	}
	t.Log("✓ Tag created")

	// === LIST ===
	t.Log("Running: vers tag list")
	out, err = testutil.RunVers(t, testutil.DefaultTimeout, "tag", "list")
	if err != nil {
		t.Fatalf("vers tag list failed: %v\nOutput:\n%s", err, out)
	}
	if !strings.Contains(out, tagName) {
		t.Fatalf("expected tag %s in list output, got:\n%s", tagName, out)
	}
	t.Log("✓ Tag listed")

	// === GET ===
	t.Logf("Running: vers tag get %s", tagName)
	out, err = testutil.RunVers(t, testutil.DefaultTimeout, "tag", "get", tagName)
	if err != nil {
		t.Fatalf("vers tag get failed: %v\nOutput:\n%s", err, out)
	}
	if !strings.Contains(out, tagName) {
		t.Fatalf("expected tag name in get output, got:\n%s", out)
	}
	if !strings.Contains(out, commitID) {
		t.Fatalf("expected commit ID in get output, got:\n%s", out)
	}
	t.Log("✓ Tag retrieved")

	// === UPDATE (description) ===
	t.Logf("Running: vers tag update %s --description 'updated desc'", tagName)
	out, err = testutil.RunVers(t, testutil.DefaultTimeout, "tag", "update", tagName, "--description", "updated desc")
	if err != nil {
		t.Fatalf("vers tag update failed: %v\nOutput:\n%s", err, out)
	}
	if !strings.Contains(out, "updated") {
		t.Fatalf("expected 'updated' in output, got:\n%s", out)
	}
	t.Log("✓ Tag updated")

	// Verify the update took effect
	out, err = testutil.RunVers(t, testutil.DefaultTimeout, "tag", "get", tagName)
	if err != nil {
		t.Fatalf("vers tag get after update failed: %v\nOutput:\n%s", err, out)
	}
	if !strings.Contains(out, "updated desc") {
		t.Fatalf("expected updated description in get output, got:\n%s", out)
	}
	t.Log("✓ Tag update verified")

	// === DELETE ===
	t.Logf("Running: vers tag delete %s", tagName)
	out, err = testutil.RunVers(t, testutil.DefaultTimeout, "tag", "delete", tagName)
	if err != nil {
		t.Fatalf("vers tag delete failed: %v\nOutput:\n%s", err, out)
	}
	if !strings.Contains(out, "deleted") {
		t.Fatalf("expected 'deleted' in output, got:\n%s", out)
	}
	t.Log("✓ Tag deleted")

	// Verify tag is gone
	out, err = testutil.RunVers(t, testutil.DefaultTimeout, "tag", "get", tagName)
	if err == nil {
		t.Fatalf("expected error getting deleted tag, got success:\n%s", out)
	}
	t.Log("✓ Deleted tag no longer accessible")

	t.Log("TestTagLifecycle completed")
}

// TestTagCreateDuplicate tests that creating a tag with an existing name fails.
func TestTagCreateDuplicate(t *testing.T) {
	t.Log("Starting TestTagCreateDuplicate...")
	testutil.TestEnv(t)
	testutil.EnsureBuilt(t)

	// Create a VM and commit
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

	tagName := testutil.UniqueAlias("dup-tag")
	t.Cleanup(func() {
		testutil.RunVers(t, testutil.DefaultTimeout, "tag", "delete", tagName)
	})

	// Create tag first time — should succeed
	out, err = testutil.RunVers(t, testutil.DefaultTimeout, "tag", "create", tagName, commitID)
	if err != nil {
		t.Fatalf("first tag create failed: %v\nOutput:\n%s", err, out)
	}

	// Create same tag again — should fail
	out, err = testutil.RunVers(t, testutil.DefaultTimeout, "tag", "create", tagName, commitID)
	if err == nil {
		t.Fatalf("expected error creating duplicate tag, got success:\n%s", out)
	}
	t.Logf("Got expected error:\n%s", out)
	t.Log("✓ Duplicate tag creation failed as expected")
}

// TestTagGetNonExistent tests that getting a non-existent tag fails gracefully.
func TestTagGetNonExistent(t *testing.T) {
	t.Log("Starting TestTagGetNonExistent...")
	testutil.TestEnv(t)
	testutil.EnsureBuilt(t)

	out, err := testutil.RunVers(t, testutil.DefaultTimeout, "tag", "get", "nonexistent-tag-xyz-999")
	if err == nil {
		t.Fatalf("expected error getting non-existent tag, got success:\n%s", out)
	}
	t.Log("✓ Non-existent tag get failed as expected")
}

// TestTagDeleteNonExistent tests that deleting a non-existent tag fails gracefully.
func TestTagDeleteNonExistent(t *testing.T) {
	t.Log("Starting TestTagDeleteNonExistent...")
	testutil.TestEnv(t)
	testutil.EnsureBuilt(t)

	out, err := testutil.RunVers(t, testutil.DefaultTimeout, "tag", "delete", "nonexistent-tag-xyz-999")
	if err == nil {
		t.Fatalf("expected error deleting non-existent tag, got success:\n%s", out)
	}
	t.Log("✓ Non-existent tag delete failed as expected")
}

// TestTagListEmpty tests that tag list works even with no tags.
func TestTagListEmpty(t *testing.T) {
	t.Log("Starting TestTagListEmpty...")
	testutil.TestEnv(t)
	testutil.EnsureBuilt(t)

	out, err := testutil.RunVers(t, testutil.DefaultTimeout, "tag", "list")
	if err != nil {
		t.Fatalf("vers tag list failed: %v\nOutput:\n%s", err, out)
	}

	if !strings.Contains(out, "Tags") && !strings.Contains(out, "tag") {
		t.Fatalf("expected tag-related output, got:\n%s", out)
	}

	t.Log("✓ Tag list works")
}
