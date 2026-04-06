package test

import (
	"strings"
	"testing"

	"github.com/hdresearch/vers-cli/test/testutil"
)

// TestRepoLifecycle exercises the full repo CRUD lifecycle against production:
// create repo → list → get → create tag → list tags → get tag → update tag → delete tag → delete repo.
// Everything is cleaned up regardless of test outcome.
func TestRepoLifecycle(t *testing.T) {
	testutil.TestEnv(t)
	testutil.EnsureBuilt(t)

	repoName := testutil.UniqueAlias("repo")

	// Always clean up the repo at end, even if something fails midway.
	t.Cleanup(func() {
		// Best-effort delete; ignore errors (repo may already be deleted).
		testutil.RunVers(t, testutil.DefaultTimeout, "repo", "delete", repoName)
	})

	// ── Create repo ──────────────────────────────────────────────
	out, err := testutil.RunVers(t, testutil.DefaultTimeout,
		"repo", "create", repoName, "-d", "integration test repo")
	if err != nil {
		t.Fatalf("repo create failed: %v\nOutput:\n%s", err, out)
	}
	if !strings.Contains(out, repoName) {
		t.Fatalf("expected repo name in output, got:\n%s", out)
	}

	// ── List repos ───────────────────────────────────────────────
	out, err = testutil.RunVers(t, testutil.DefaultTimeout, "repo", "list")
	if err != nil {
		t.Fatalf("repo list failed: %v\nOutput:\n%s", err, out)
	}
	if !strings.Contains(out, repoName) {
		t.Fatalf("expected %s in list output, got:\n%s", repoName, out)
	}

	// ── List repos (quiet) ───────────────────────────────────────
	out, err = testutil.RunVers(t, testutil.DefaultTimeout, "repo", "list", "-q")
	if err != nil {
		t.Fatalf("repo list -q failed: %v\nOutput:\n%s", err, out)
	}
	found := false
	for _, line := range strings.Split(strings.TrimSpace(out), "\n") {
		if strings.TrimSpace(line) == repoName {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("expected %s in quiet list output, got:\n%s", repoName, out)
	}

	// ── List repos (json) ────────────────────────────────────────
	out, err = testutil.RunVers(t, testutil.DefaultTimeout, "repo", "list", "--format", "json")
	if err != nil {
		t.Fatalf("repo list --format json failed: %v\nOutput:\n%s", err, out)
	}
	if !strings.Contains(out, repoName) {
		t.Fatalf("expected %s in json output, got:\n%s", repoName, out)
	}

	// ── Get repo ─────────────────────────────────────────────────
	out, err = testutil.RunVers(t, testutil.DefaultTimeout, "repo", "get", repoName)
	if err != nil {
		t.Fatalf("repo get failed: %v\nOutput:\n%s", err, out)
	}
	if !strings.Contains(out, repoName) {
		t.Fatalf("expected %s in get output, got:\n%s", repoName, out)
	}
	if !strings.Contains(out, "integration test repo") {
		t.Fatalf("expected description in get output, got:\n%s", out)
	}

	// ── We need a commit to create a tag. Create a VM, commit it, clean up. ──
	out, err = testutil.RunVers(t, testutil.DefaultTimeout, "run")
	if err != nil {
		t.Fatalf("vers run failed: %v\nOutput:\n%s", err, out)
	}
	vmID, err := testutil.ParseVMID(out)
	if err != nil {
		t.Fatalf("failed to parse VM ID: %v\nOutput:\n%s", err, out)
	}
	testutil.RegisterVMCleanup(t, vmID, true)

	out, err = testutil.RunVers(t, testutil.DefaultTimeout, "commit", "create", vmID)
	if err != nil {
		t.Fatalf("commit create failed: %v\nOutput:\n%s", err, out)
	}
	commitID := parseCommitID(t, out)

	// ── Create tag in repo ───────────────────────────────────────
	out, err = testutil.RunVers(t, testutil.DefaultTimeout,
		"repo", "tag", "create", repoName, "v1", commitID, "-d", "first release")
	if err != nil {
		t.Fatalf("repo tag create failed: %v\nOutput:\n%s", err, out)
	}
	if !strings.Contains(out, repoName+":v1") {
		t.Fatalf("expected reference %s:v1 in output, got:\n%s", repoName, out)
	}

	// ── List tags ────────────────────────────────────────────────
	out, err = testutil.RunVers(t, testutil.DefaultTimeout,
		"repo", "tag", "list", repoName)
	if err != nil {
		t.Fatalf("repo tag list failed: %v\nOutput:\n%s", err, out)
	}
	if !strings.Contains(out, "v1") {
		t.Fatalf("expected v1 in tag list output, got:\n%s", out)
	}

	// ── List tags (quiet) ────────────────────────────────────────
	out, err = testutil.RunVers(t, testutil.DefaultTimeout,
		"repo", "tag", "list", repoName, "-q")
	if err != nil {
		t.Fatalf("repo tag list -q failed: %v\nOutput:\n%s", err, out)
	}
	if strings.TrimSpace(out) != "v1" {
		t.Fatalf("expected 'v1' in quiet tag list, got:\n%s", out)
	}

	// ── Get tag ──────────────────────────────────────────────────
	out, err = testutil.RunVers(t, testutil.DefaultTimeout,
		"repo", "tag", "get", repoName, "v1")
	if err != nil {
		t.Fatalf("repo tag get failed: %v\nOutput:\n%s", err, out)
	}
	if !strings.Contains(out, commitID) {
		t.Fatalf("expected commit ID in tag get output, got:\n%s", out)
	}
	if !strings.Contains(out, "first release") {
		t.Fatalf("expected description in tag get output, got:\n%s", out)
	}

	// ── Update tag description ───────────────────────────────────
	out, err = testutil.RunVers(t, testutil.DefaultTimeout,
		"repo", "tag", "update", repoName, "v1", "-d", "updated release")
	if err != nil {
		t.Fatalf("repo tag update failed: %v\nOutput:\n%s", err, out)
	}

	// ── Delete tag ───────────────────────────────────────────────
	out, err = testutil.RunVers(t, testutil.DefaultTimeout,
		"repo", "tag", "delete", repoName, "v1")
	if err != nil {
		t.Fatalf("repo tag delete failed: %v\nOutput:\n%s", err, out)
	}
	if !strings.Contains(out, "deleted") {
		t.Fatalf("expected 'deleted' in output, got:\n%s", out)
	}

	// ── Verify tag is gone ───────────────────────────────────────
	out, err = testutil.RunVers(t, testutil.DefaultTimeout,
		"repo", "tag", "list", repoName)
	if err != nil {
		t.Fatalf("repo tag list after delete failed: %v\nOutput:\n%s", err, out)
	}
	if strings.Contains(out, "v1") && !strings.Contains(out, "No tags found") {
		t.Fatalf("expected v1 to be gone from tag list, got:\n%s", out)
	}

	// ── Delete repo ──────────────────────────────────────────────
	out, err = testutil.RunVers(t, testutil.DefaultTimeout, "repo", "delete", repoName)
	if err != nil {
		t.Fatalf("repo delete failed: %v\nOutput:\n%s", err, out)
	}
	if !strings.Contains(out, "deleted") {
		t.Fatalf("expected 'deleted' in output, got:\n%s", out)
	}

	// ── Verify repo is gone ──────────────────────────────────────
	out, err = testutil.RunVers(t, testutil.DefaultTimeout, "repo", "list", "-q")
	if err != nil {
		t.Fatalf("repo list after delete failed: %v\nOutput:\n%s", err, out)
	}
	if strings.Contains(out, repoName) {
		t.Fatalf("expected %s to be gone from list, got:\n%s", repoName, out)
	}
}

// parseCommitID extracts a commit ID from `vers commit create` output.
// Expected format:
//
//	✓ Committed VM '<vm-id>'
//	Commit ID: <commit-id>
func parseCommitID(t *testing.T, output string) string {
	t.Helper()
	for _, line := range strings.Split(output, "\n") {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "Commit ID:") {
			parts := strings.SplitN(line, ":", 2)
			if len(parts) == 2 {
				id := strings.TrimSpace(parts[1])
				if id != "" {
					return id
				}
			}
		}
		// Fallback: a bare UUID on its own line
		if len(line) == 36 && strings.Count(line, "-") == 4 {
			return line
		}
	}
	t.Fatalf("could not parse commit ID from output:\n%s", output)
	return ""
}
