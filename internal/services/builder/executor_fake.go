package builder

import (
	"context"
	"errors"
	"fmt"
	"io"
	"sort"
	"sync"
)

// FakeExecutor is an in-memory Executor for testing the build loop.
//
// It maintains a set of "VMs" (opaque string ids) and "commits" (opaque
// string ids), and records every call it receives so tests can assert on
// the exact sequence. Commands are handled by RunFunc (a matcher function
// supplied by the test); unconfigured commands succeed with exit code 0
// unless RunFunc is set, in which case RunFunc decides.
//
// The fake is safe for serial use only — the builder runs single-threaded
// so we don't attempt real concurrency hardening.
type FakeExecutor struct {
	mu sync.Mutex

	// State
	NextVMNum     int
	NextCommitNum int
	LiveVMs       map[string]bool
	Commits       map[string]bool // commits that exist server-side
	Tags          map[string]string
	// CommitParent[new] = old; so we can reason about lineage.
	CommitParent map[string]string
	// VMBase[vmID] = commit it was restored from ("" for scratch).
	VMBase map[string]string

	// Call log, newest last.
	Calls []Call

	// Knobs
	RunFunc            func(cmd []string, env map[string]string, workdir string) (exitCode int, stdout, stderr string, err error)
	UploadFunc         func(vmID, local, remote string, recursive bool) error
	FailNextCommit     bool
	FailNextRestore    bool
	MissingCommits     map[string]bool // commits RestoreFromCommit rejects as missing
}

// Call is a single recorded executor call.
type Call struct {
	Op        string // "NewVM", "RestoreFromCommit", "Run", "Upload", "Commit", "DeleteVM", "CreateTag", "ResolveTag"
	VmID      string
	CommitID  string
	TagName   string
	Cmd       []string
	Env       map[string]string
	Workdir   string
	LocalSrc  string
	RemoteDst string
	Recursive bool
	Spec      VMSpec
}

// NewFake returns an empty FakeExecutor.
func NewFake() *FakeExecutor {
	return &FakeExecutor{
		LiveVMs:        map[string]bool{},
		Commits:        map[string]bool{},
		Tags:           map[string]string{},
		CommitParent:   map[string]string{},
		VMBase:         map[string]string{},
		MissingCommits: map[string]bool{},
	}
}

// Seed creates a pre-existing commit (used to simulate FROM <tag>).
func (f *FakeExecutor) Seed(commitID string) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.Commits[commitID] = true
}

// SeedTag associates a tag with a pre-existing commit.
func (f *FakeExecutor) SeedTag(tag, commitID string) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.Tags[tag] = commitID
	f.Commits[commitID] = true
}

// OpsOnly returns the call log projected to just the Op names, which is
// the most common assertion in tests.
func (f *FakeExecutor) OpsOnly() []string {
	f.mu.Lock()
	defer f.mu.Unlock()
	out := make([]string, len(f.Calls))
	for i, c := range f.Calls {
		out[i] = c.Op
	}
	return out
}

// CountOp returns how many times a given op appeared.
func (f *FakeExecutor) CountOp(name string) int {
	n := 0
	for _, o := range f.OpsOnly() {
		if o == name {
			n++
		}
	}
	return n
}

// CommitOrder returns the sequence of commit ids issued via Commit(), in
// order. Useful for verifying the layer chain.
func (f *FakeExecutor) CommitOrder() []string {
	f.mu.Lock()
	defer f.mu.Unlock()
	var out []string
	for _, c := range f.Calls {
		if c.Op == "Commit" && c.CommitID != "" {
			out = append(out, c.CommitID)
		}
	}
	return out
}

// -- Executor -------------------------------------------------------------

func (f *FakeExecutor) NewVM(ctx context.Context, spec VMSpec) (string, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.NextVMNum++
	id := fmt.Sprintf("vm-%d", f.NextVMNum)
	f.LiveVMs[id] = true
	f.VMBase[id] = ""
	f.Calls = append(f.Calls, Call{Op: "NewVM", VmID: id, Spec: spec})
	return id, nil
}

func (f *FakeExecutor) RestoreFromCommit(ctx context.Context, commitID string) (string, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.FailNextRestore {
		f.FailNextRestore = false
		f.Calls = append(f.Calls, Call{Op: "RestoreFromCommit", CommitID: commitID})
		return "", errors.New("restore failed (injected)")
	}
	if f.MissingCommits[commitID] {
		f.Calls = append(f.Calls, Call{Op: "RestoreFromCommit", CommitID: commitID})
		return "", fmt.Errorf("commit %s not found", commitID)
	}
	f.NextVMNum++
	id := fmt.Sprintf("vm-%d", f.NextVMNum)
	f.LiveVMs[id] = true
	f.VMBase[id] = commitID
	f.Commits[commitID] = true
	f.Calls = append(f.Calls, Call{Op: "RestoreFromCommit", CommitID: commitID, VmID: id})
	return id, nil
}

func (f *FakeExecutor) ResolveTag(ctx context.Context, name string) (string, bool) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.Calls = append(f.Calls, Call{Op: "ResolveTag", TagName: name})
	c, ok := f.Tags[name]
	return c, ok
}

func (f *FakeExecutor) Run(ctx context.Context, vmID string, cmd []string, env map[string]string, workdir string, stdout, stderr io.Writer) (int, error) {
	f.mu.Lock()
	if !f.LiveVMs[vmID] {
		f.mu.Unlock()
		return -1, fmt.Errorf("run against dead VM %q", vmID)
	}
	// Copy env into a sorted map snapshot for test determinism.
	envCopy := map[string]string{}
	for k, v := range env {
		envCopy[k] = v
	}
	cmdCopy := append([]string(nil), cmd...)
	f.Calls = append(f.Calls, Call{Op: "Run", VmID: vmID, Cmd: cmdCopy, Env: envCopy, Workdir: workdir})
	runFn := f.RunFunc
	f.mu.Unlock()

	if runFn == nil {
		return 0, nil
	}
	code, out, errOut, err := runFn(cmdCopy, envCopy, workdir)
	if out != "" {
		_, _ = io.WriteString(stdout, out)
	}
	if errOut != "" {
		_, _ = io.WriteString(stderr, errOut)
	}
	return code, err
}

func (f *FakeExecutor) Upload(ctx context.Context, vmID, local, remote string, recursive bool) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	if !f.LiveVMs[vmID] {
		return fmt.Errorf("upload against dead VM %q", vmID)
	}
	f.Calls = append(f.Calls, Call{Op: "Upload", VmID: vmID, LocalSrc: local, RemoteDst: remote, Recursive: recursive})
	if f.UploadFunc != nil {
		return f.UploadFunc(vmID, local, remote, recursive)
	}
	return nil
}

func (f *FakeExecutor) Commit(ctx context.Context, vmID string) (string, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.FailNextCommit {
		f.FailNextCommit = false
		f.Calls = append(f.Calls, Call{Op: "Commit", VmID: vmID})
		return "", errors.New("commit failed (injected)")
	}
	if !f.LiveVMs[vmID] {
		return "", fmt.Errorf("commit against dead VM %q", vmID)
	}
	f.NextCommitNum++
	id := fmt.Sprintf("c-%d", f.NextCommitNum)
	f.Commits[id] = true
	f.CommitParent[id] = f.VMBase[vmID]
	f.Calls = append(f.Calls, Call{Op: "Commit", VmID: vmID, CommitID: id})
	return id, nil
}

func (f *FakeExecutor) CreateTag(ctx context.Context, name, commitID string) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.Calls = append(f.Calls, Call{Op: "CreateTag", TagName: name, CommitID: commitID})
	f.Tags[name] = commitID
	return nil
}

func (f *FakeExecutor) DeleteVM(ctx context.Context, vmID string) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.Calls = append(f.Calls, Call{Op: "DeleteVM", VmID: vmID})
	delete(f.LiveVMs, vmID)
	return nil
}

// LiveVMList returns the currently-live VM ids, sorted.
func (f *FakeExecutor) LiveVMList() []string {
	f.mu.Lock()
	defer f.mu.Unlock()
	out := make([]string, 0, len(f.LiveVMs))
	for id := range f.LiveVMs {
		out = append(out, id)
	}
	sort.Strings(out)
	return out
}
