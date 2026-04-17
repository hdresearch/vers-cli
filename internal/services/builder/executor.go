package builder

import (
	"context"
	"io"
)

// Executor is the narrow set of remote operations the build loop depends on.
//
// Everything the builder needs from the Vers backend goes through this
// interface: the builder itself does zero direct SDK / SSH / orchestrator
// calls. That lets us (a) unit-test the full build loop with a fake, and
// (b) swap the backend later (e.g. a local Firecracker executor) without
// touching build logic.
type Executor interface {
	// NewVM creates a fresh VM per spec and waits until it's running.
	// Used for FROM scratch.
	NewVM(ctx context.Context, spec VMSpec) (vmID string, err error)

	// RestoreFromCommit creates a VM from an existing commit and waits
	// until it's running. Used for FROM <commit>, cache hits, and
	// post-cache VM switches.
	RestoreFromCommit(ctx context.Context, commitID string) (vmID string, err error)

	// ResolveTag returns the commit id for a named tag, or ("", false)
	// if the tag does not exist. Errors (network / auth) are collapsed
	// into "not found" — the caller will treat the input as a commit id.
	ResolveTag(ctx context.Context, name string) (commitID string, ok bool)

	// Run executes a command on the VM, streaming stdout and stderr to
	// the provided writers. Returns the command's exit code.
	Run(ctx context.Context, vmID string, cmd []string, env map[string]string, workdir string, stdout, stderr io.Writer) (exitCode int, err error)

	// Upload transfers a single local path (file or directory) to the VM.
	// `recursive` must be true for directories.
	Upload(ctx context.Context, vmID, localAbs, remote string, recursive bool) error

	// Commit snapshots the VM and returns the new commit id.
	Commit(ctx context.Context, vmID string) (commitID string, err error)

	// CreateTag points a named tag at a commit.
	CreateTag(ctx context.Context, name, commitID string) error

	// DeleteVM removes a VM. Errors are returned but should usually be
	// surfaced as warnings by the caller (teardown is best-effort).
	DeleteVM(ctx context.Context, vmID string) error
}

// VMSpec is the sizing/config for a fresh VM (FROM scratch).
type VMSpec struct {
	MemSizeMib  int64
	VcpuCount   int64
	FsSizeVmMib int64
	RootfsName  string // optional
	KernelName  string // optional
}
