// Package jobs implements a durable, append-only JSONL ledger for tracking
// `--wait` invocations across vers-cli runs (F14 Phase 1: journaling only).
//
// Phase 1 ships journaling: every `--wait` command appends a "submitted" entry
// when it dispatches the API call, and a "complete" or "failed" entry when the
// handler returns. Resumption of in-flight jobs is intentionally out of scope
// (Phase 2).
//
// Writes are best-effort: a write or directory failure must NOT cause the
// invoking command to fail. Callers should ignore the error returned from
// Submit/Complete/Fail in normal flow (or log it to stderr at most).
package jobs

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

// Status values for ledger entries.
const (
	StatusSubmitted = "submitted"
	StatusComplete  = "complete"
	StatusFailed    = "failed"
)

// Entry is a single ledger record. Each line in the JSONL file is one Entry.
// Entries with the same ID represent state transitions of the same job; the
// latest line wins when collapsing for reads.
type Entry struct {
	ID          string     `json:"id"`
	Kind        string     `json:"kind"`
	Command     string     `json:"command"`
	Args        []string   `json:"args"`
	StartedAt   time.Time  `json:"started_at"`
	CompletedAt *time.Time `json:"completed_at,omitempty"`
	DurationMs  *int64     `json:"duration_ms,omitempty"`
	Status      string     `json:"status"`
	ResultID    string     `json:"result_id,omitempty"`
	Error       string     `json:"error,omitempty"`
}

// Submission is the input for Submit. Args should not include the program
// name; Command is a short, human-readable invocation summary like
// "vers run --wait".
type Submission struct {
	Kind    string
	Command string
	Args    []string
}

// dirOverride, when non-empty, takes precedence over the VERS_JOBS_DIR
// environment variable and the default (~/.vers). Tests use SetDir.
var dirOverride string

// SetDir overrides the directory the ledger is stored in for the current
// process. Pass "" to clear the override (env var / default re-applies).
// Intended for tests.
func SetDir(dir string) { dirOverride = dir }

// Dir returns the resolved ledger directory.
//
// Precedence: explicit override (SetDir) > VERS_JOBS_DIR env > $HOME/.vers.
func Dir() (string, error) {
	if dirOverride != "" {
		return dirOverride, nil
	}
	if d := os.Getenv("VERS_JOBS_DIR"); d != "" {
		return d, nil
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".vers"), nil
}

// Path returns the full path to the JSONL ledger file.
func Path() (string, error) {
	d, err := Dir()
	if err != nil {
		return "", err
	}
	return filepath.Join(d, "jobs.jsonl"), nil
}

// Submit appends a new "submitted" entry and returns the assigned job id.
// On any I/O failure it returns an empty string and a non-nil error; the
// caller should treat that as best-effort and continue regardless.
func Submit(s Submission) (string, error) {
	id, err := newJobID()
	if err != nil {
		return "", err
	}
	e := Entry{
		ID:        id,
		Kind:      s.Kind,
		Command:   s.Command,
		Args:      append([]string{}, s.Args...),
		StartedAt: time.Now().UTC(),
		Status:    StatusSubmitted,
	}
	if err := Append(e); err != nil {
		return id, err
	}
	return id, nil
}

// Complete appends a terminal "complete" entry referencing an existing id.
// If id is empty (e.g. Submit failed), Complete is a no-op.
func Complete(id, resultID string) error {
	if id == "" {
		return nil
	}
	prev, ok, err := latestByID(id)
	if err != nil {
		return err
	}
	now := time.Now().UTC()
	e := Entry{
		ID:          id,
		Status:      StatusComplete,
		CompletedAt: &now,
		ResultID:    resultID,
	}
	if ok {
		copyMeta(&e, prev)
		dur := now.Sub(prev.StartedAt).Milliseconds()
		e.DurationMs = &dur
	}
	return Append(e)
}

// Fail appends a terminal "failed" entry. If id is empty, Fail is a no-op.
// A nil err is recorded as an empty error string but still marked failed.
func Fail(id string, err error) error {
	if id == "" {
		return nil
	}
	prev, ok, lerr := latestByID(id)
	if lerr != nil {
		return lerr
	}
	now := time.Now().UTC()
	msg := ""
	if err != nil {
		msg = err.Error()
	}
	e := Entry{
		ID:          id,
		Status:      StatusFailed,
		CompletedAt: &now,
		Error:       msg,
	}
	if ok {
		copyMeta(&e, prev)
		dur := now.Sub(prev.StartedAt).Milliseconds()
		e.DurationMs = &dur
	}
	return Append(e)
}

// copyMeta carries forward identity-stable fields from an earlier entry into
// a terminal entry so the latest line is self-contained.
func copyMeta(dst *Entry, src Entry) {
	dst.Kind = src.Kind
	dst.Command = src.Command
	dst.Args = append([]string{}, src.Args...)
	dst.StartedAt = src.StartedAt
}

// Append writes a single entry as one JSON line to the ledger file. The
// directory is created if missing.
func Append(e Entry) error {
	dir, err := Dir()
	if err != nil {
		return err
	}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	path := filepath.Join(dir, "jobs.jsonl")
	f, err := os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
	if err != nil {
		return err
	}
	defer f.Close()
	b, err := json.Marshal(e)
	if err != nil {
		return err
	}
	b = append(b, '\n')
	_, err = f.Write(b)
	return err
}

// readAll reads every entry in file order (append order). Missing file
// returns (nil, nil).
func readAll() ([]Entry, error) {
	path, err := Path()
	if err != nil {
		return nil, err
	}
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	var out []Entry
	for _, line := range strings.Split(string(data), "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		var e Entry
		if err := json.Unmarshal([]byte(line), &e); err != nil {
			// Skip malformed lines rather than failing the whole read.
			continue
		}
		out = append(out, e)
	}
	return out, nil
}

// Latest returns the latest state per job id, ordered by StartedAt descending
// (most recent first). Optional statusFilter, when non-empty, restricts to
// entries with matching status.
func Latest(statusFilter string) ([]Entry, error) {
	all, err := readAll()
	if err != nil {
		return nil, err
	}
	by := make(map[string]Entry, len(all))
	for _, e := range all {
		prev, ok := by[e.ID]
		if !ok {
			by[e.ID] = e
			continue
		}
		// Later line wins. Prefer terminal status; otherwise keep the
		// most-recent line by file order (which equals append order).
		if isTerminal(e.Status) || !isTerminal(prev.Status) {
			by[e.ID] = e
		}
	}
	out := make([]Entry, 0, len(by))
	for _, e := range by {
		if statusFilter != "" && e.Status != statusFilter {
			continue
		}
		out = append(out, e)
	}
	sort.Slice(out, func(i, j int) bool {
		return out[i].StartedAt.After(out[j].StartedAt)
	})
	return out, nil
}

// Get returns the latest state of one job id.
func Get(id string) (Entry, bool, error) {
	return latestByID(id)
}

func latestByID(id string) (Entry, bool, error) {
	all, err := readAll()
	if err != nil {
		return Entry{}, false, err
	}
	var found Entry
	ok := false
	for _, e := range all {
		if e.ID != id {
			continue
		}
		if !ok {
			found = e
			ok = true
			continue
		}
		if isTerminal(e.Status) || !isTerminal(found.Status) {
			found = e
		}
	}
	return found, ok, nil
}

func isTerminal(s string) bool {
	return s == StatusComplete || s == StatusFailed
}

// PruneOptions configures Prune.
type PruneOptions struct {
	OlderThan time.Duration // entries with StartedAt older than now-OlderThan are removed
	All       bool          // when true, removes everything regardless of OlderThan
	DryRun    bool          // when true, no file is written
	Now       time.Time     // optional clock override; zero value uses time.Now().UTC()
}

// PruneResult summarises a prune call.
type PruneResult struct {
	Pruned int
	Kept   int
}

// Prune removes ledger entries by age (or all of them) and returns counts.
// In DryRun mode the ledger file is untouched.
//
// Pruning operates on the latest-state-per-id collapse: a job is kept iff its
// latest entry is younger than the cutoff. When kept, all of its entries are
// preserved to maintain the audit trail.
func Prune(opts PruneOptions) (PruneResult, error) {
	all, err := readAll()
	if err != nil {
		return PruneResult{}, err
	}
	if len(all) == 0 {
		return PruneResult{}, nil
	}

	now := opts.Now
	if now.IsZero() {
		now = time.Now().UTC()
	}

	// Collapse to latest state per id to decide which ids to keep.
	latest := make(map[string]Entry)
	for _, e := range all {
		prev, ok := latest[e.ID]
		if !ok || isTerminal(e.Status) || !isTerminal(prev.Status) {
			latest[e.ID] = e
		}
	}

	keepIDs := make(map[string]bool, len(latest))
	prunedIDs := 0
	keptIDs := 0
	for id, e := range latest {
		if opts.All {
			prunedIDs++
			continue
		}
		ref := e.StartedAt
		if e.CompletedAt != nil {
			ref = *e.CompletedAt
		}
		if now.Sub(ref) > opts.OlderThan {
			prunedIDs++
			continue
		}
		keepIDs[id] = true
		keptIDs++
	}

	if opts.DryRun {
		return PruneResult{Pruned: prunedIDs, Kept: keptIDs}, nil
	}

	// Rewrite the file with only kept entries.
	path, err := Path()
	if err != nil {
		return PruneResult{}, err
	}
	if opts.All || keptIDs == 0 {
		if err := os.Remove(path); err != nil && !os.IsNotExist(err) {
			return PruneResult{}, err
		}
		return PruneResult{Pruned: prunedIDs, Kept: 0}, nil
	}

	tmp := path + ".tmp"
	f, err := os.Create(tmp)
	if err != nil {
		return PruneResult{}, err
	}
	enc := json.NewEncoder(f)
	for _, e := range all {
		if !keepIDs[e.ID] {
			continue
		}
		if err := enc.Encode(e); err != nil {
			f.Close()
			os.Remove(tmp)
			return PruneResult{}, err
		}
	}
	if err := f.Close(); err != nil {
		os.Remove(tmp)
		return PruneResult{}, err
	}
	if err := os.Rename(tmp, path); err != nil {
		os.Remove(tmp)
		return PruneResult{}, err
	}
	return PruneResult{Pruned: prunedIDs, Kept: keptIDs}, nil
}

// ParseDuration accepts the small set we surface to users: "7d", "24h",
// "30m", etc. Anything time.ParseDuration accepts is also accepted; the
// special "Nd" suffix converts to N*24h.
func ParseDuration(s string) (time.Duration, error) {
	s = strings.TrimSpace(s)
	if s == "" {
		return 0, fmt.Errorf("duration is empty")
	}
	if strings.HasSuffix(s, "d") {
		var days int
		if _, err := fmt.Sscanf(strings.TrimSuffix(s, "d"), "%d", &days); err != nil {
			return 0, fmt.Errorf("invalid duration %q: must be like 7d, 24h, 30m", s)
		}
		if days < 0 {
			return 0, fmt.Errorf("duration must be non-negative: %q", s)
		}
		return time.Duration(days) * 24 * time.Hour, nil
	}
	d, err := time.ParseDuration(s)
	if err != nil {
		return 0, fmt.Errorf("invalid duration %q: must be like 7d, 24h, 30m", s)
	}
	return d, nil
}

func newJobID() (string, error) {
	var b [5]byte
	if _, err := rand.Read(b[:]); err != nil {
		return "", err
	}
	return "job_" + hex.EncodeToString(b[:]), nil
}
