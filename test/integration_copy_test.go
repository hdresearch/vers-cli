package test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/hdresearch/vers-cli/test/testutil"
)

// TestCopyWithNonExistentVM verifies copy fails early with a clear error when VM is unknown.
func TestCopyWithNonExistentVM(t *testing.T) {
	testutil.TestEnv(t)
	testutil.EnsureBuilt(t)

	// Create a temporary local file to act as upload source
	dir := t.TempDir()
	src := filepath.Join(dir, "test-upload.txt")
	if err := os.WriteFile(src, []byte("hello"), 0644); err != nil {
		t.Fatalf("failed to create temp file: %v", err)
	}

	out, err := testutil.RunVers(t, testutil.DefaultTimeout, "copy", "non-existent-vm", src, "/tmp/test.txt")
	if err == nil {
		t.Fatalf("expected failure for non-existent VM, got success. Output:\n%s", out)
	}
	if !containsAny(out, "failed to get VM information", "not found", "could not resolve") {
		t.Fatalf("expected clear VM resolution failure, got:\n%s", out)
	}
}

func containsAny(s string, subs ...string) bool {
	for _, sub := range subs {
		if sub != "" && strings.Contains(s, sub) {
			return true
		}
	}
	return false
}
