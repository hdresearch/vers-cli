// Package builder executes a parsed Dockerfile against a Vers VM.
//
// Strategy:
//
//  1. FROM creates the initial VM (either fresh via vm.NewRoot, or restored
//     from a commit/tag via vm.RestoreFromCommit).
//  2. Each subsequent instruction is executed against that VM. After the
//     step succeeds we `vm.Commit` to produce a "layer" commit id which is
//     both the cache value and the parent for the next step's cache key.
//  3. Per-step cache keys combine (parent commit, normalized instruction,
//     optional content hash). If a key hits, we skip execution and branch
//     from the cached commit instead.
//  4. The final commit id is the build output.
//
// The executor is deliberately stateless across runs aside from the on-disk
// .vers/buildcache.json map. It owns a single VM for the life of the build
// and tears it down at the end (unless --keep).
package builder

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/hdresearch/vers-cli/internal/app"
	"github.com/hdresearch/vers-cli/internal/dockerfile"
	delsvc "github.com/hdresearch/vers-cli/internal/services/deletion"
	vmSvc "github.com/hdresearch/vers-cli/internal/services/vm"
	sshutil "github.com/hdresearch/vers-cli/internal/ssh"
	"github.com/hdresearch/vers-cli/internal/utils"
	vers "github.com/hdresearch/vers-sdk-go"
)

// Options controls a build.
type Options struct {
	Instructions []dockerfile.Instruction
	Context      *BuildContext
	BuildArgs    map[string]string

	// FROM scratch machine sizing. Required when FROM scratch is used.
	MemSizeMib  int64
	VcpuCount   int64
	FsSizeVmMib int64
	RootfsName  string // optional; defaults to "default" server-side
	KernelName  string

	NoCache bool
	Keep    bool // if true, the builder VM is not deleted at the end
	Tag     string // optional vers tag to create on the final commit
}

// Result is the outcome of a successful build.
type Result struct {
	FinalCommitID string
	BuilderVmID   string // set if Keep=true
	StepCount     int
	CachedCount   int
	Tag           string // populated if Tag was requested and created
	Cmd           []string
	Entrypoint    []string
	ExposedPorts  []string
	Labels        map[string]string
}

// state carries builder state across instructions.
type state struct {
	env     map[string]string
	argVals map[string]string // values for declared ARGs
	declaredArgs map[string]bool
	workdir string
	user    string
	labels  map[string]string
	cmd     []string
	cmdShell string
	entrypoint []string
	entrypointShell string
	exposed []string
}

// Build runs the instructions and returns the final commit id.
//
// Progress messages are written to a.IO.Err so JSON output on stdout stays
// clean. The caller is expected to surface the returned error.
func Build(ctx context.Context, a *app.App, opts Options) (*Result, error) {
	if len(opts.Instructions) == 0 {
		return nil, fmt.Errorf("empty Dockerfile: no instructions")
	}

	// v1: reject multi-stage
	stages := 0
	for _, ins := range opts.Instructions {
		if ins.Kind == dockerfile.KindFrom {
			stages++
		}
	}
	if stages != 1 {
		return nil, fmt.Errorf("multi-stage builds are not supported yet (found %d FROM instructions)", stages)
	}

	first := opts.Instructions[0]
	if first.Kind != dockerfile.KindFrom {
		return nil, fmt.Errorf("Dockerfile must start with FROM")
	}

	cache := LoadCache()
	if opts.NoCache {
		cache.Entries = map[string]string{}
	}

	st := &state{
		env:          map[string]string{},
		argVals:      map[string]string{},
		declaredArgs: map[string]bool{},
		labels:       map[string]string{},
	}
	for k, v := range opts.BuildArgs {
		st.argVals[k] = v
	}

	// ---- FROM ----------------------------------------------------------------
	var vmID, baseCommit string
	var err error
	vmID, baseCommit, err = runFrom(ctx, a, first.From, opts)
	if err != nil {
		return nil, err
	}
	// Ensure cleanup if we fail mid-build and --keep wasn't set
	cleanup := func() {
		if opts.Keep || vmID == "" {
			return
		}
		ctx2, cancel := context.WithTimeout(context.Background(), a.Timeouts.APIShort)
		defer cancel()
		_, _ = delsvc.DeleteVM(ctx2, a.Client, vmID)
	}
	defer func() {
		if err != nil {
			cleanup()
		}
	}()

	fmt.Fprintf(a.IO.Err, "Step 1/%d : %s\n", len(opts.Instructions), first.Raw)
	if baseCommit != "" {
		fmt.Fprintf(a.IO.Err, " ---> using base commit %s\n", short(baseCommit))
	} else {
		fmt.Fprintf(a.IO.Err, " ---> built fresh VM %s\n", short(vmID))
	}

	parentCommit := baseCommit
	res := &Result{StepCount: len(opts.Instructions)}

	// ---- Loop over remaining instructions -----------------------------------
	for i := 1; i < len(opts.Instructions); i++ {
		ins := opts.Instructions[i]
		fmt.Fprintf(a.IO.Err, "Step %d/%d : %s\n", i+1, len(opts.Instructions), ins.Raw)

		// Pure-metadata instructions: apply to state, do not commit a new layer.
		// These still affect *future* layers' cache keys via applyMetaExtras.
		if handled, metaErr := applyMeta(st, ins); handled {
			if metaErr != nil {
				err = metaErr
				return nil, err
			}
			fmt.Fprintf(a.IO.Err, " ---> metadata\n")
			continue
		}

		// Build cache key.
		key, extras, kerr := cacheKeyFor(ins, st, opts.Context, parentCommit)
		if kerr != nil {
			err = kerr
			return nil, err
		}
		_ = extras

		if !opts.NoCache {
			if cached := cache.Get(key); cached != "" {
				// Fast-forward: tear down current VM, branch from the cached commit.
				newVM, switchErr := switchToCommit(ctx, a, vmID, cached)
				if switchErr != nil {
					// Cache hit but commit missing: fall through to real execution.
					fmt.Fprintf(a.IO.Err, " ---> cache entry stale (%v), rebuilding\n", switchErr)
				} else {
					vmID = newVM
					parentCommit = cached
					res.CachedCount++
					fmt.Fprintf(a.IO.Err, " ---> using cached layer %s\n", short(cached))
					continue
				}
			}
		}

		// Execute.
		execErr := runStep(ctx, a, vmID, st, ins, opts.Context)
		if execErr != nil {
			err = fmt.Errorf("step %d (%s): %w", i+1, ins.Kind, execErr)
			return nil, err
		}

		// Commit the layer.
		commitResp, cerr := a.Client.Vm.Commit(ctx, vmID, vers.VmCommitParams{})
		if cerr != nil {
			err = fmt.Errorf("commit after step %d: %w", i+1, cerr)
			return nil, err
		}
		parentCommit = commitResp.CommitID
		cache.Put(key, parentCommit)
		cache.Save()
		fmt.Fprintf(a.IO.Err, " ---> %s\n", short(parentCommit))
	}

	if parentCommit == "" {
		// Only a FROM — no further commits. Commit once so we have a result.
		commitResp, cerr := a.Client.Vm.Commit(ctx, vmID, vers.VmCommitParams{})
		if cerr != nil {
			err = fmt.Errorf("commit initial state: %w", cerr)
			return nil, err
		}
		parentCommit = commitResp.CommitID
	}

	res.FinalCommitID = parentCommit
	res.Cmd = st.cmd
	if len(st.cmd) == 0 && st.cmdShell != "" {
		res.Cmd = []string{"sh", "-c", st.cmdShell}
	}
	res.Entrypoint = st.entrypoint
	if len(st.entrypoint) == 0 && st.entrypointShell != "" {
		res.Entrypoint = []string{"sh", "-c", st.entrypointShell}
	}
	res.Labels = st.labels
	res.ExposedPorts = st.exposed

	// Optional tag.
	if opts.Tag != "" {
		_, terr := a.Client.CommitTags.New(ctx, vers.CommitTagNewParams{
			CreateTagRequest: vers.CreateTagRequestParam{
				TagName:  vers.F(opts.Tag),
				CommitID: vers.F(parentCommit),
			},
		})
		if terr != nil {
			// Tagging failure shouldn't kill the build; surface as warning.
			fmt.Fprintf(a.IO.Err, "warning: failed to create tag %q: %v\n", opts.Tag, terr)
		} else {
			res.Tag = opts.Tag
		}
	}

	// Teardown unless --keep.
	if opts.Keep {
		res.BuilderVmID = vmID
	} else {
		ctx2, cancel := context.WithTimeout(context.Background(), a.Timeouts.APIShort)
		defer cancel()
		if _, derr := delsvc.DeleteVM(ctx2, a.Client, vmID); derr != nil {
			fmt.Fprintf(a.IO.Err, "warning: failed to delete builder VM %s: %v\n", vmID, derr)
		}
	}

	return res, nil
}

// runFrom materializes the base VM. Returns (vmID, baseCommitID).
// baseCommitID is "" when starting from scratch.
func runFrom(ctx context.Context, a *app.App, f *dockerfile.FromInstr, opts Options) (string, string, error) {
	if f.As != "" {
		// Parse accepted it but executor doesn't support named stages yet.
		return "", "", fmt.Errorf("FROM ... AS is not supported in single-stage v1")
	}
	if strings.EqualFold(f.Ref, "scratch") {
		if opts.MemSizeMib == 0 || opts.VcpuCount == 0 || opts.FsSizeVmMib == 0 {
			return "", "", fmt.Errorf("FROM scratch requires --mem-size, --vcpu-count, and --fs-size-vm-mib")
		}
		cfg := vers.NewRootRequestVmConfigParam{
			MemSizeMib: vers.F(opts.MemSizeMib),
			VcpuCount:  vers.F(opts.VcpuCount),
			FsSizeMib:  vers.F(opts.FsSizeVmMib),
		}
		if opts.RootfsName != "" {
			cfg.ImageName = vers.F(opts.RootfsName)
		}
		if opts.KernelName != "" {
			cfg.KernelName = vers.F(opts.KernelName)
		}
		resp, err := a.Client.Vm.NewRoot(ctx, vers.VmNewRootParams{
			NewRootRequest: vers.NewRootRequestParam{VmConfig: vers.F(cfg)},
		})
		if err != nil {
			return "", "", fmt.Errorf("FROM scratch: %w", err)
		}
		if err := utils.WaitForRunning(ctx, a.Client, resp.VmID); err != nil {
			return "", "", fmt.Errorf("FROM scratch: wait: %w", err)
		}
		return resp.VmID, "", nil
	}

	// Otherwise: try as tag first, then as commit id.
	commitID := f.Ref
	if tag, terr := a.Client.CommitTags.Get(ctx, f.Ref); terr == nil && tag != nil && tag.CommitID != "" {
		commitID = tag.CommitID
	}
	resp, err := a.Client.Vm.RestoreFromCommit(ctx, vers.VmRestoreFromCommitParams{
		VmFromCommitRequest: vers.VmFromCommitRequestParam{CommitID: vers.F(commitID)},
	})
	if err != nil {
		return "", "", fmt.Errorf("FROM %s: %w", f.Ref, err)
	}
	if err := utils.WaitForRunning(ctx, a.Client, resp.VmID); err != nil {
		return "", "", fmt.Errorf("FROM %s: wait: %w", f.Ref, err)
	}
	return resp.VmID, commitID, nil
}

// switchToCommit tears down the current VM and restores a fresh one from
// the given cached commit. Returns the new VM id.
func switchToCommit(ctx context.Context, a *app.App, oldVM, commitID string) (string, error) {
	resp, err := a.Client.Vm.RestoreFromCommit(ctx, vers.VmRestoreFromCommitParams{
		VmFromCommitRequest: vers.VmFromCommitRequestParam{CommitID: vers.F(commitID)},
	})
	if err != nil {
		return "", err
	}
	if err := utils.WaitForRunning(ctx, a.Client, resp.VmID); err != nil {
		_, _ = delsvc.DeleteVM(ctx, a.Client, resp.VmID)
		return "", err
	}
	// Delete the old VM in the background-ish (still inside ctx).
	if oldVM != "" {
		_, _ = delsvc.DeleteVM(ctx, a.Client, oldVM)
	}
	return resp.VmID, nil
}

// applyMeta handles instructions that only update builder state, with no
// corresponding VM mutation. Returns handled=true if it took care of the
// instruction.
func applyMeta(st *state, ins dockerfile.Instruction) (handled bool, err error) {
	switch ins.Kind {
	case dockerfile.KindArg:
		st.declaredArgs[ins.Arg.Name] = true
		if _, present := st.argVals[ins.Arg.Name]; !present && ins.Arg.HasDef {
			st.argVals[ins.Arg.Name] = ins.Arg.Default
		}
		return true, nil
	case dockerfile.KindEnv:
		for _, kv := range ins.Env.Pairs {
			st.env[kv.Key] = expand(kv.Value, st)
		}
		return true, nil
	case dockerfile.KindLabel:
		for _, kv := range ins.Label.Pairs {
			st.labels[kv.Key] = expand(kv.Value, st)
		}
		return true, nil
	case dockerfile.KindWorkdir:
		w := expand(ins.Workdir.Value, st)
		if filepath.IsAbs(w) || st.workdir == "" {
			st.workdir = w
		} else {
			st.workdir = filepath.Join(st.workdir, w)
		}
		return true, nil
	case dockerfile.KindUser:
		st.user = expand(ins.User.Value, st)
		return true, nil
	case dockerfile.KindCmd:
		st.cmd = ins.Cmd.Exec
		st.cmdShell = ins.Cmd.Shell
		return true, nil
	case dockerfile.KindEntrypoint:
		st.entrypoint = ins.Entrypoint.Exec
		st.entrypointShell = ins.Entrypoint.Shell
		return true, nil
	case dockerfile.KindExpose:
		st.exposed = append(st.exposed, ins.Expose.Ports...)
		return true, nil
	}
	return false, nil
}

// cacheKeyFor produces a cache key for a *materializing* instruction
// (RUN / COPY / ADD). Metadata instructions don't create layers.
func cacheKeyFor(ins dockerfile.Instruction, st *state, bc *BuildContext, parentCommit string) (string, []string, error) {
	extras := metaExtras(st)
	switch ins.Kind {
	case dockerfile.KindRun:
		// The expanded shell form bakes current ENV/WORKDIR/USER into the key.
		cmd := runCommandLine(ins.Run, st)
		return CacheKey(parentCommit, "RUN "+cmd, extras...), extras, nil
	case dockerfile.KindCopy, dockerfile.KindAdd:
		if ins.Copy.From != "" {
			return "", nil, fmt.Errorf("COPY --from is not supported in v1")
		}
		hash, err := bc.HashSources(ins.Copy.Sources)
		if err != nil {
			return "", nil, err
		}
		extras = append(extras, "tree="+hash, "chown="+ins.Copy.Chown)
		return CacheKey(parentCommit, ins.Raw, extras...), extras, nil
	default:
		return "", nil, fmt.Errorf("unhandled instruction %s", ins.Kind)
	}
}

// metaExtras returns a sorted, deterministic slice encoding the environment
// captured in the builder state. Any future RUN or COPY depends on this.
func metaExtras(st *state) []string {
	var out []string
	if st.workdir != "" {
		out = append(out, "wd="+st.workdir)
	}
	if st.user != "" {
		out = append(out, "user="+st.user)
	}
	keys := make([]string, 0, len(st.env))
	for k := range st.env {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	for _, k := range keys {
		out = append(out, "env="+k+"="+st.env[k])
	}
	return out
}

// runStep executes RUN or COPY against the live VM.
func runStep(ctx context.Context, a *app.App, vmID string, st *state, ins dockerfile.Instruction, bc *BuildContext) error {
	switch ins.Kind {
	case dockerfile.KindRun:
		return runRun(ctx, a, vmID, st, ins.Run)
	case dockerfile.KindCopy, dockerfile.KindAdd:
		return runCopy(ctx, a, vmID, st, ins.Copy, bc)
	}
	return fmt.Errorf("runStep: unhandled instruction %s", ins.Kind)
}

func runRun(ctx context.Context, a *app.App, vmID string, st *state, r *dockerfile.RunInstr) error {
	cmd := runCommand(r, st)
	body, err := vmSvc.ExecStream(ctx, vmID, vmSvc.ExecRequest{
		Command:    cmd,
		Env:        copyMap(st.env),
		WorkingDir: st.workdir,
	})
	if err != nil {
		return err
	}
	defer body.Close()
	code, err := streamOutput(body, a.IO.Out, a.IO.Err)
	if err != nil {
		return err
	}
	if code != 0 {
		return fmt.Errorf("RUN exited with code %d", code)
	}
	return nil
}

// runCommand materializes the actual argv passed to exec, respecting USER.
func runCommand(r *dockerfile.RunInstr, st *state) []string {
	if len(r.Exec) > 0 {
		return maybeWrapUser(r.Exec, st)
	}
	return maybeWrapUser([]string{"bash", "-c", r.Shell}, st)
}

// runCommandLine is a flat, stable text representation of the command for
// cache-key purposes only.
func runCommandLine(r *dockerfile.RunInstr, st *state) string {
	cmd := runCommand(r, st)
	return strings.Join(cmd, "\x00")
}

func maybeWrapUser(cmd []string, st *state) []string {
	if st.user == "" {
		return cmd
	}
	joined := utilsShellJoin(cmd)
	return []string{"sudo", "-u", st.user, "bash", "-c", joined}
}

func runCopy(ctx context.Context, a *app.App, vmID string, st *state, c *dockerfile.CopyInstr, bc *BuildContext) error {
	if c.From != "" {
		return fmt.Errorf("COPY --from is not supported in v1")
	}
	info, err := vmSvc.GetConnectInfo(ctx, a.Client, vmID)
	if err != nil {
		return fmt.Errorf("connect info: %w", err)
	}
	client := sshutil.NewClient(info.Host, info.KeyPath, info.VMDomain)

	dest := c.Dest
	if !filepath.IsAbs(dest) {
		// Relative dest is resolved against WORKDIR (or / if unset).
		base := st.workdir
		if base == "" {
			base = "/"
		}
		dest = filepath.Join(base, dest)
	}

	// If multiple sources, destination must be a directory (docker rule).
	destIsDir := strings.HasSuffix(c.Dest, "/") || len(c.Sources) > 1

	// Ensure destination exists.
	mkdirTarget := dest
	if !destIsDir {
		mkdirTarget = filepath.Dir(dest)
	}
	if mkdirTarget != "" && mkdirTarget != "/" {
		if err := execSimple(ctx, vmID, []string{"mkdir", "-p", mkdirTarget}); err != nil {
			return fmt.Errorf("mkdir %s: %w", mkdirTarget, err)
		}
	}

	for _, srcSpec := range c.Sources {
		entries, err := bc.ResolveSource(srcSpec)
		if err != nil {
			return err
		}
		if len(entries) == 0 {
			return fmt.Errorf("no files matched source %q (all ignored?)", srcSpec)
		}
		// Compute the single filesystem path we hand to sftp.Upload.
		srcAbs := filepath.Join(bc.Root, filepath.FromSlash(strings.TrimPrefix(srcSpec, "./")))
		st2, serr := os.Stat(srcAbs)
		if serr != nil {
			return serr
		}
		var remoteTarget string
		if destIsDir {
			remoteTarget = filepath.Join(dest, filepath.Base(srcAbs))
		} else {
			remoteTarget = dest
		}
		recursive := st2.IsDir()
		if err := client.Upload(ctx, srcAbs, remoteTarget, recursive); err != nil {
			return fmt.Errorf("upload %s: %w", srcSpec, err)
		}
	}

	// Optional --chown
	if c.Chown != "" {
		if err := execSimple(ctx, vmID, []string{"chown", "-R", c.Chown, dest}); err != nil {
			return fmt.Errorf("chown: %w", err)
		}
	}
	return nil
}

// execSimple runs a command on the VM and errors if it exits non-zero,
// discarding output. Used for internal plumbing (mkdir, chown).
func execSimple(ctx context.Context, vmID string, cmd []string) error {
	body, err := vmSvc.ExecStream(ctx, vmID, vmSvc.ExecRequest{Command: cmd})
	if err != nil {
		return err
	}
	defer body.Close()
	code, err := streamOutput(body, io.Discard, io.Discard)
	if err != nil {
		return err
	}
	if code != 0 {
		return fmt.Errorf("%s exited %d", strings.Join(cmd, " "), code)
	}
	return nil
}

func copyMap(m map[string]string) map[string]string {
	if m == nil {
		return nil
	}
	out := make(map[string]string, len(m))
	for k, v := range m {
		out[k] = v
	}
	return out
}

// expand performs lightweight $VAR / ${VAR} substitution across ENV +
// build args, matching Docker's semantics for ENV/LABEL/WORKDIR/USER.
func expand(s string, st *state) string {
	if !strings.ContainsAny(s, "$") {
		return s
	}
	lookup := func(name string) (string, bool) {
		if v, ok := st.argVals[name]; ok && st.declaredArgs[name] {
			return v, true
		}
		if v, ok := st.env[name]; ok {
			return v, true
		}
		return "", false
	}
	var b strings.Builder
	i := 0
	for i < len(s) {
		if s[i] != '$' {
			b.WriteByte(s[i])
			i++
			continue
		}
		// ${NAME}
		if i+1 < len(s) && s[i+1] == '{' {
			end := strings.IndexByte(s[i+2:], '}')
			if end < 0 {
				b.WriteByte(s[i])
				i++
				continue
			}
			name := s[i+2 : i+2+end]
			v, _ := lookup(name)
			b.WriteString(v)
			i += 2 + end + 1
			continue
		}
		// $NAME
		j := i + 1
		for j < len(s) && (isAlnum(s[j]) || s[j] == '_') {
			j++
		}
		if j == i+1 {
			b.WriteByte(s[i])
			i++
			continue
		}
		name := s[i+1 : j]
		v, _ := lookup(name)
		b.WriteString(v)
		i = j
	}
	return b.String()
}

func isAlnum(c byte) bool {
	return (c >= '0' && c <= '9') || (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z')
}

func short(id string) string {
	if len(id) <= 12 {
		return id
	}
	return id[:12]
}

// utilsShellJoin is intentionally here (not using utils.ShellJoin) so the
// package can be tested standalone without dragging the whole utils tree.
func utilsShellJoin(args []string) string {
	var b strings.Builder
	for i, a := range args {
		if i > 0 {
			b.WriteByte(' ')
		}
		if strings.ContainsAny(a, " \t\"'\\$`") {
			b.WriteByte('\'')
			b.WriteString(strings.ReplaceAll(a, "'", `'\''`))
			b.WriteByte('\'')
		} else {
			b.WriteString(a)
		}
	}
	return b.String()
}
