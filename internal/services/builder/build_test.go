package builder

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/hdresearch/vers-cli/internal/dockerfile"
)

// --- Test harness ------------------------------------------------------------

// buildHarness constructs a valid build context (one file) and parses the
// given Dockerfile text. Returns an Options ready to hand to Build with a
// FakeExecutor attached.
func buildHarness(t *testing.T, df string) (*FakeExecutor, Options, *bytes.Buffer) {
	t.Helper()
	dir := t.TempDir()
	// Default: a single file the Dockerfiles below can COPY.
	if err := os.WriteFile(filepath.Join(dir, "file.txt"), []byte("hello"), 0644); err != nil {
		t.Fatal(err)
	}
	instrs, err := dockerfile.Parse(strings.NewReader(df))
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	bc, err := LoadContext(dir)
	if err != nil {
		t.Fatal(err)
	}
	fake := NewFake()
	stderr := &bytes.Buffer{}
	return fake, Options{
		Instructions: instrs,
		Context:      bc,
		Exec:         fake,
		Stderr:       stderr,
		MemSizeMib:   512,
		VcpuCount:    1,
		FsSizeVmMib:  1024,
	}, stderr
}

// --- Happy-path scenarios ----------------------------------------------------

func TestBuild_FromScratchSimpleChain(t *testing.T) {
	fake, opts, _ := buildHarness(t, "FROM scratch\nRUN echo hi\nRUN echo bye\n")
	res, err := Build(context.Background(), nil, opts)
	if err != nil {
		t.Fatalf("Build: %v", err)
	}
	// Two RUN steps -> two Commit calls. No cache hits.
	if res.StepCount != 3 {
		t.Errorf("steps=%d want 3", res.StepCount)
	}
	if res.CachedCount != 0 {
		t.Errorf("cached=%d want 0", res.CachedCount)
	}
	if fake.CountOp("Commit") != 2 {
		t.Errorf("commits=%d want 2", fake.CountOp("Commit"))
	}
	if fake.CountOp("NewVM") != 1 {
		t.Errorf("NewVM=%d want 1", fake.CountOp("NewVM"))
	}
	// Builder VM gets deleted at end (no --keep).
	if fake.CountOp("DeleteVM") != 1 {
		t.Errorf("DeleteVM=%d want 1", fake.CountOp("DeleteVM"))
	}
	if len(fake.LiveVMList()) != 0 {
		t.Errorf("expected no live VMs, got %v", fake.LiveVMList())
	}
	// Final commit id should be the last Commit()
	order := fake.CommitOrder()
	if len(order) == 0 || res.FinalCommitID != order[len(order)-1] {
		t.Errorf("final commit mismatch: res=%s order=%v", res.FinalCommitID, order)
	}
}

func TestBuild_MetadataDoesNotCommit(t *testing.T) {
	fake, opts, _ := buildHarness(t, `FROM scratch
ENV FOO=bar
WORKDIR /app
USER root
LABEL key=val
EXPOSE 80
CMD ["echo"]
RUN true
`)
	res, err := Build(context.Background(), nil, opts)
	if err != nil {
		t.Fatal(err)
	}
	// Only one RUN should produce a commit.
	if fake.CountOp("Commit") != 1 {
		t.Errorf("commits=%d want 1", fake.CountOp("Commit"))
	}
	if res.Cmd == nil || res.Cmd[0] != "echo" {
		t.Errorf("cmd=%+v", res.Cmd)
	}
	if res.ExposedPorts[0] != "80" {
		t.Errorf("exposed=%+v", res.ExposedPorts)
	}
	if res.Labels["key"] != "val" {
		t.Errorf("labels=%+v", res.Labels)
	}
}

func TestBuild_FromTag(t *testing.T) {
	fake, opts, _ := buildHarness(t, "FROM prod\nRUN true\n")
	fake.SeedTag("prod", "seed-commit")
	res, err := Build(context.Background(), nil, opts)
	if err != nil {
		t.Fatal(err)
	}
	if fake.CountOp("NewVM") != 0 {
		t.Errorf("unexpected NewVM call: %+v", fake.OpsOnly())
	}
	if fake.CountOp("RestoreFromCommit") != 1 {
		t.Errorf("want one RestoreFromCommit, got %+v", fake.OpsOnly())
	}
	// The restore should have targeted the tag's commit.
	for _, c := range fake.Calls {
		if c.Op == "RestoreFromCommit" && c.CommitID != "seed-commit" {
			t.Errorf("restored wrong commit: %q", c.CommitID)
		}
	}
	if res.FinalCommitID == "" {
		t.Error("expected a final commit id")
	}
}

func TestBuild_FromCommitIDPassthrough(t *testing.T) {
	// No tag named "abc-123", so Build should restore that id as-is.
	fake, opts, _ := buildHarness(t, "FROM abc-123\nRUN true\n")
	_, err := Build(context.Background(), nil, opts)
	if err != nil {
		t.Fatal(err)
	}
	foundRestore := false
	for _, c := range fake.Calls {
		if c.Op == "RestoreFromCommit" {
			foundRestore = true
			if c.CommitID != "abc-123" {
				t.Errorf("restored %q, want abc-123", c.CommitID)
			}
		}
	}
	if !foundRestore {
		t.Error("expected RestoreFromCommit call")
	}
}

// --- Cache behaviour --------------------------------------------------------

func TestBuild_CacheHitSkipsRunAndSwitchesVM(t *testing.T) {
	dir := t.TempDir()
	t.Chdir(dir)
	if err := os.MkdirAll(".vers", 0755); err != nil {
		t.Fatal(err)
	}

	// First build populates the cache.
	fake1, opts1, _ := buildHarness(t, "FROM scratch\nRUN echo hi\n")
	res1, err := Build(context.Background(), nil, opts1)
	if err != nil {
		t.Fatal(err)
	}
	if fake1.CountOp("Run") == 0 {
		t.Fatal("expected first build to execute RUN")
	}

	// Second build with same Dockerfile and context should hit the cache.
	// Seed the second fake with the commit the first build produced so the
	// cache-hit path can restore it.
	fake2, opts2, _ := buildHarness(t, "FROM scratch\nRUN echo hi\n")
	fake2.Seed(res1.FinalCommitID)
	res2, err := Build(context.Background(), nil, opts2)
	if err != nil {
		t.Fatal(err)
	}
	if res2.CachedCount != 1 {
		t.Errorf("cached=%d want 1", res2.CachedCount)
	}
	// No RUN should have been issued (the cache hit skips execution).
	for _, c := range fake2.Calls {
		if c.Op == "Run" {
			t.Errorf("unexpected Run on cache hit: %+v", c.Cmd)
		}
	}
	// And no Commit, since we reused the cached commit.
	if n := fake2.CountOp("Commit"); n != 0 {
		t.Errorf("commits on cache hit=%d want 0", n)
	}
	// A cache hit branches from the cached commit.
	if fake2.CountOp("RestoreFromCommit") != 1 {
		t.Errorf("RestoreFromCommit=%d want 1 (branch from cache)", fake2.CountOp("RestoreFromCommit"))
	}
	// The initial scratch VM must have been deleted when we switched over.
	if fake2.CountOp("DeleteVM") != 2 {
		// one for the switch, one for teardown
		t.Errorf("DeleteVM=%d want 2 (switch + teardown)", fake2.CountOp("DeleteVM"))
	}
	// Final result identical to first build's final commit.
	if res2.FinalCommitID != res1.FinalCommitID {
		t.Errorf("final mismatch: first=%s second=%s", res1.FinalCommitID, res2.FinalCommitID)
	}
}

func TestBuild_NoCacheBypassesLookup(t *testing.T) {
	dir := t.TempDir()
	t.Chdir(dir)
	if err := os.MkdirAll(".vers", 0755); err != nil {
		t.Fatal(err)
	}

	// Prime the cache.
	_, opts1, _ := buildHarness(t, "FROM scratch\nRUN echo hi\n")
	if _, err := Build(context.Background(), nil, opts1); err != nil {
		t.Fatal(err)
	}

	// Second build with NoCache: must actually run again.
	fake2, opts2, _ := buildHarness(t, "FROM scratch\nRUN echo hi\n")
	opts2.NoCache = true
	res, err := Build(context.Background(), nil, opts2)
	if err != nil {
		t.Fatal(err)
	}
	if res.CachedCount != 0 {
		t.Errorf("cached=%d, expected 0 with --no-cache", res.CachedCount)
	}
	if fake2.CountOp("Run") == 0 {
		t.Error("expected Run with --no-cache")
	}
}

func TestBuild_StaleCacheFallsThroughToExecution(t *testing.T) {
	dir := t.TempDir()
	t.Chdir(dir)
	if err := os.MkdirAll(".vers", 0755); err != nil {
		t.Fatal(err)
	}

	// Prime the cache.
	_, opts1, _ := buildHarness(t, "FROM scratch\nRUN echo hi\n")
	res1, err := Build(context.Background(), nil, opts1)
	if err != nil {
		t.Fatal(err)
	}

	// Second build: cache still points at a commit, but the server no
	// longer has it. The builder should fall through and re-execute.
	fake2, opts2, stderr := buildHarness(t, "FROM scratch\nRUN echo hi\n")
	fake2.MissingCommits = map[string]bool{res1.FinalCommitID: true}
	res2, err := Build(context.Background(), nil, opts2)
	if err != nil {
		t.Fatalf("Build: %v", err)
	}
	if res2.CachedCount != 0 {
		t.Errorf("cached=%d want 0 when server lost the commit", res2.CachedCount)
	}
	if fake2.CountOp("Run") == 0 {
		t.Error("expected fallback Run")
	}
	if !strings.Contains(stderr.String(), "stale") {
		t.Errorf("expected progress note about stale cache, got:\n%s", stderr.String())
	}
}

// --- Failure and teardown ---------------------------------------------------

func TestBuild_RunFailureDeletesBuilder(t *testing.T) {
	fake, opts, _ := buildHarness(t, "FROM scratch\nRUN false\n")
	fake.RunFunc = func(cmd []string, env map[string]string, workdir string) (int, string, string, error) {
		return 2, "", "boom", nil
	}
	res, err := Build(context.Background(), nil, opts)
	if err == nil {
		t.Fatal("expected error from failing RUN")
	}
	if res != nil {
		t.Errorf("expected nil result on error, got %+v", res)
	}
	if fake.CountOp("DeleteVM") != 1 {
		t.Errorf("DeleteVM=%d want 1 (teardown on failure)", fake.CountOp("DeleteVM"))
	}
	if len(fake.LiveVMList()) != 0 {
		t.Errorf("VM leaked: %v", fake.LiveVMList())
	}
}

func TestBuild_KeepLeavesBuilderAlive(t *testing.T) {
	fake, opts, _ := buildHarness(t, "FROM scratch\nRUN true\n")
	opts.Keep = true
	res, err := Build(context.Background(), nil, opts)
	if err != nil {
		t.Fatal(err)
	}
	if fake.CountOp("DeleteVM") != 0 {
		t.Errorf("DeleteVM=%d want 0 with --keep", fake.CountOp("DeleteVM"))
	}
	if res.BuilderVmID == "" {
		t.Error("expected BuilderVmID in result with --keep")
	}
	if !contains(fake.LiveVMList(), res.BuilderVmID) {
		t.Errorf("builder VM should be alive, have %v", fake.LiveVMList())
	}
}

func TestBuild_KeepOnFailureStillLeaves(t *testing.T) {
	fake, opts, _ := buildHarness(t, "FROM scratch\nRUN false\n")
	opts.Keep = true
	fake.RunFunc = func(cmd []string, env map[string]string, workdir string) (int, string, string, error) {
		return 2, "", "", nil
	}
	_, err := Build(context.Background(), nil, opts)
	if err == nil {
		t.Fatal("expected error")
	}
	if fake.CountOp("DeleteVM") != 0 {
		t.Errorf("--keep should leave VM even on failure, got DeleteVM=%d", fake.CountOp("DeleteVM"))
	}
	if len(fake.LiveVMList()) != 1 {
		t.Errorf("expected exactly one live VM, got %v", fake.LiveVMList())
	}
}

func TestBuild_CommitFailurePropagates(t *testing.T) {
	fake, opts, _ := buildHarness(t, "FROM scratch\nRUN true\n")
	fake.FailNextCommit = true
	_, err := Build(context.Background(), nil, opts)
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "commit") {
		t.Errorf("want commit error, got: %v", err)
	}
	if fake.CountOp("DeleteVM") != 1 {
		t.Errorf("DeleteVM=%d want 1 (cleanup)", fake.CountOp("DeleteVM"))
	}
}

// --- Tagging ----------------------------------------------------------------

func TestBuild_TagsFinalCommit(t *testing.T) {
	fake, opts, _ := buildHarness(t, "FROM scratch\nRUN true\n")
	opts.Tag = "myapp:prod"
	res, err := Build(context.Background(), nil, opts)
	if err != nil {
		t.Fatal(err)
	}
	if res.Tag != "myapp:prod" {
		t.Errorf("result.Tag=%q want myapp:prod", res.Tag)
	}
	if fake.Tags["myapp:prod"] != res.FinalCommitID {
		t.Errorf("tag points at wrong commit: %q vs %q", fake.Tags["myapp:prod"], res.FinalCommitID)
	}
}

// --- COPY wiring ------------------------------------------------------------

func TestBuild_CopyIssuesMkdirAndUpload(t *testing.T) {
	fake, opts, _ := buildHarness(t, "FROM scratch\nCOPY file.txt /dst/file.txt\n")
	if _, err := Build(context.Background(), nil, opts); err != nil {
		t.Fatal(err)
	}
	var sawMkdir, sawUpload bool
	for _, c := range fake.Calls {
		if c.Op == "Run" && len(c.Cmd) >= 2 && c.Cmd[0] == "mkdir" && c.Cmd[1] == "-p" {
			sawMkdir = true
		}
		if c.Op == "Upload" && c.RemoteDst == "/dst/file.txt" && !c.Recursive {
			sawUpload = true
		}
	}
	if !sawMkdir {
		t.Errorf("expected mkdir -p call, got ops: %v", fake.OpsOnly())
	}
	if !sawUpload {
		t.Errorf("expected Upload to /dst/file.txt, got calls:\n%+v", fake.Calls)
	}
}

func TestBuild_CopyWithChownRunsChown(t *testing.T) {
	fake, opts, _ := buildHarness(t, "FROM scratch\nCOPY --chown=node:node file.txt /dst\n")
	if _, err := Build(context.Background(), nil, opts); err != nil {
		t.Fatal(err)
	}
	sawChown := false
	for _, c := range fake.Calls {
		if c.Op == "Run" && len(c.Cmd) > 0 && c.Cmd[0] == "chown" {
			sawChown = true
			if len(c.Cmd) < 3 || c.Cmd[2] != "node:node" {
				t.Errorf("chown args wrong: %+v", c.Cmd)
			}
		}
	}
	if !sawChown {
		t.Error("expected chown Run call")
	}
}

// --- helpers ----------------------------------------------------------------

func contains(ss []string, want string) bool {
	for _, s := range ss {
		if s == want {
			return true
		}
	}
	return false
}
