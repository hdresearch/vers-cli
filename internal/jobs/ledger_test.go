package jobs

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func setup(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	SetDir(dir)
	t.Cleanup(func() { SetDir("") })
	return dir
}

func TestSubmitCompleteRoundtrip(t *testing.T) {
	dir := setup(t)

	id, err := Submit(Submission{Kind: "vm.run", Command: "vers run --wait", Args: []string{"--vm-alias", "x"}})
	if err != nil {
		t.Fatalf("Submit: %v", err)
	}
	if !strings.HasPrefix(id, "job_") {
		t.Fatalf("expected job_ prefix, got %q", id)
	}
	if err := Complete(id, "vm-abc"); err != nil {
		t.Fatalf("Complete: %v", err)
	}

	// File should have two lines.
	data, err := os.ReadFile(filepath.Join(dir, "jobs.jsonl"))
	if err != nil {
		t.Fatalf("ReadFile: %v", err)
	}
	if got := strings.Count(strings.TrimSpace(string(data)), "\n") + 1; got != 2 {
		t.Fatalf("expected 2 lines, got %d (%s)", got, data)
	}

	got, ok, err := Get(id)
	if err != nil || !ok {
		t.Fatalf("Get: ok=%v err=%v", ok, err)
	}
	if got.Status != StatusComplete {
		t.Fatalf("expected complete, got %q", got.Status)
	}
	if got.ResultID != "vm-abc" {
		t.Fatalf("expected result id carried, got %q", got.ResultID)
	}
	if got.Kind != "vm.run" || got.Command != "vers run --wait" {
		t.Fatalf("metadata not carried: %+v", got)
	}
	if got.DurationMs == nil {
		t.Fatalf("expected duration to be set")
	}
}

func TestFail(t *testing.T) {
	setup(t)
	id, err := Submit(Submission{Kind: "vm.branch", Command: "vers branch --wait"})
	if err != nil {
		t.Fatalf("Submit: %v", err)
	}
	if err := Fail(id, errors.New("boom")); err != nil {
		t.Fatalf("Fail: %v", err)
	}
	got, ok, err := Get(id)
	if err != nil || !ok {
		t.Fatalf("Get: ok=%v err=%v", ok, err)
	}
	if got.Status != StatusFailed || got.Error != "boom" {
		t.Fatalf("unexpected entry: %+v", got)
	}
}

func TestLatestCollapseAndOrder(t *testing.T) {
	setup(t)
	a, _ := Submit(Submission{Kind: "vm.run", Command: "vers run"})
	time.Sleep(2 * time.Millisecond)
	b, _ := Submit(Submission{Kind: "vm.branch", Command: "vers branch"})
	time.Sleep(2 * time.Millisecond)
	c, _ := Submit(Submission{Kind: "vm.deploy", Command: "vers deploy"})

	if err := Complete(a, "vm-a"); err != nil {
		t.Fatal(err)
	}
	if err := Fail(b, errors.New("nope")); err != nil {
		t.Fatal(err)
	}
	// c stays submitted.

	all, err := Latest("")
	if err != nil {
		t.Fatalf("Latest: %v", err)
	}
	if len(all) != 3 {
		t.Fatalf("expected 3 jobs, got %d", len(all))
	}
	// Order: most-recent StartedAt first.
	if all[0].ID != c || all[1].ID != b || all[2].ID != a {
		t.Fatalf("order wrong: %+v", []string{all[0].ID, all[1].ID, all[2].ID})
	}
	// Verify collapse: a should be complete, b failed, c submitted.
	statusByID := map[string]string{}
	for _, e := range all {
		statusByID[e.ID] = e.Status
	}
	if statusByID[a] != StatusComplete || statusByID[b] != StatusFailed || statusByID[c] != StatusSubmitted {
		t.Fatalf("collapse wrong: %+v", statusByID)
	}

	// Filter by status.
	failed, _ := Latest(StatusFailed)
	if len(failed) != 1 || failed[0].ID != b {
		t.Fatalf("status filter failed: %+v", failed)
	}
}

func TestPruneByAge(t *testing.T) {
	dir := setup(t)
	// Manually craft entries with old timestamps so Prune has something to remove.
	old := time.Now().UTC().Add(-30 * 24 * time.Hour)
	oldDone := old.Add(time.Second)
	dur := int64(1000)
	mustAppend(t, Entry{ID: "job_old", Kind: "vm.run", Command: "vers run", StartedAt: old, Status: StatusSubmitted})
	mustAppend(t, Entry{ID: "job_old", Kind: "vm.run", Command: "vers run", StartedAt: old, CompletedAt: &oldDone, DurationMs: &dur, Status: StatusComplete, ResultID: "vm-old"})

	freshID, err := Submit(Submission{Kind: "vm.branch", Command: "vers branch"})
	if err != nil {
		t.Fatal(err)
	}
	if err := Complete(freshID, "vm-new"); err != nil {
		t.Fatal(err)
	}

	res, err := Prune(PruneOptions{OlderThan: 7 * 24 * time.Hour, DryRun: true})
	if err != nil {
		t.Fatalf("Prune dry-run: %v", err)
	}
	if res.Pruned != 1 || res.Kept != 1 {
		t.Fatalf("dry-run counts: %+v", res)
	}
	// Dry-run must not modify file.
	before, _ := os.ReadFile(filepath.Join(dir, "jobs.jsonl"))

	res, err = Prune(PruneOptions{OlderThan: 7 * 24 * time.Hour})
	if err != nil {
		t.Fatalf("Prune: %v", err)
	}
	if res.Pruned != 1 || res.Kept != 1 {
		t.Fatalf("counts: %+v", res)
	}
	after, _ := os.ReadFile(filepath.Join(dir, "jobs.jsonl"))
	if len(after) >= len(before) {
		t.Fatalf("expected file to shrink: before=%d after=%d", len(before), len(after))
	}

	if _, ok, _ := Get("job_old"); ok {
		t.Fatalf("expected job_old to be pruned")
	}
	if _, ok, _ := Get(freshID); !ok {
		t.Fatalf("expected fresh job to survive prune")
	}
}

func TestPruneAll(t *testing.T) {
	dir := setup(t)
	id, _ := Submit(Submission{Kind: "vm.run", Command: "vers run"})
	_ = Complete(id, "vm-x")

	res, err := Prune(PruneOptions{All: true})
	if err != nil {
		t.Fatalf("Prune all: %v", err)
	}
	if res.Kept != 0 || res.Pruned != 1 {
		t.Fatalf("counts: %+v", res)
	}
	if _, err := os.Stat(filepath.Join(dir, "jobs.jsonl")); !os.IsNotExist(err) {
		t.Fatalf("expected ledger removed, stat err=%v", err)
	}
}

func TestParseDuration(t *testing.T) {
	cases := []struct {
		in   string
		want time.Duration
		err  bool
	}{
		{"7d", 7 * 24 * time.Hour, false},
		{"24h", 24 * time.Hour, false},
		{"30m", 30 * time.Minute, false},
		{"0d", 0, false},
		{"", 0, true},
		{"banana", 0, true},
		{"-1d", 0, true},
	}
	for _, tc := range cases {
		got, err := ParseDuration(tc.in)
		if tc.err {
			if err == nil {
				t.Errorf("ParseDuration(%q) expected error, got %v", tc.in, got)
			}
			continue
		}
		if err != nil {
			t.Errorf("ParseDuration(%q) unexpected error: %v", tc.in, err)
			continue
		}
		if got != tc.want {
			t.Errorf("ParseDuration(%q) = %v, want %v", tc.in, got, tc.want)
		}
	}
}

func TestNoLedgerFile(t *testing.T) {
	setup(t)
	all, err := Latest("")
	if err != nil {
		t.Fatalf("Latest: %v", err)
	}
	if len(all) != 0 {
		t.Fatalf("expected empty list, got %d", len(all))
	}
	if _, ok, err := Get("job_missing"); err != nil || ok {
		t.Fatalf("Get on empty ledger: ok=%v err=%v", ok, err)
	}
}

func TestEmptyIDIsNoop(t *testing.T) {
	dir := setup(t)
	if err := Complete("", ""); err != nil {
		t.Fatal(err)
	}
	if err := Fail("", errors.New("x")); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(dir, "jobs.jsonl")); !os.IsNotExist(err) {
		t.Fatalf("ledger should not have been created, stat err=%v", err)
	}
}

func mustAppend(t *testing.T, e Entry) {
	t.Helper()
	if err := Append(e); err != nil {
		t.Fatalf("Append: %v", err)
	}
}
