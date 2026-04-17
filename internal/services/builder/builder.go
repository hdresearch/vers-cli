// Package builder executes a parsed Dockerfile against a Vers VM.
//
// Strategy:
//
//  1. FROM creates the initial VM (either fresh via Executor.NewVM, or
//     restored from a commit/tag via Executor.RestoreFromCommit).
//  2. Each subsequent instruction is executed against that VM. After the
//     step succeeds we Executor.Commit to produce a "layer" commit id
//     which is both the cache value and the parent for the next step's
//     cache key.
//  3. Per-step cache keys combine (parent commit, normalized instruction,
//     optional content hash). If a key hits, we skip execution and
//     branch from the cached commit instead.
//  4. The final commit id is the build output.
//
// All remote work flows through the Executor interface. The production
// Executor is returned by NewRealExecutor; tests inject a fake.
//
// The build is stateful only on disk via .vers/buildcache.json. It owns a
// single VM for the life of the build and tears it down at the end
// (unless Options.Keep is set).
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
	RootfsName  string
	KernelName  string

	NoCache bool
	Keep    bool // if true, the builder VM is not deleted at the end
	Tag     string // optional vers tag to create on the final commit

	// Injected dependencies. If Exec is nil, Build uses the App-backed
	// real executor. If Stderr is nil, a.IO.Err is used.
	Exec   Executor
	Stderr io.Writer
}

// Result is the outcome of a successful build.
type Result struct {
	FinalCommitID string
	BuilderVmID   string
	StepCount     int
	CachedCount   int
	Tag           string
	Cmd           []string
	Entrypoint    []string
	ExposedPorts  []string
	Labels        map[string]string
}

// state carries builder state across instructions.
type state struct {
	env             map[string]string
	argVals         map[string]string
	declaredArgs    map[string]bool
	workdir         string
	user            string
	labels          map[string]string
	cmd             []string
	cmdShell        string
	entrypoint      []string
	entrypointShell string
	exposed         []string
}

// Build runs the instructions and returns the final commit id.
//
// Progress messages go to opts.Stderr (or a.IO.Err if nil). RUN output goes
// to a.IO.Out / a.IO.Err so users see their build output as it streams.
func Build(ctx context.Context, a *app.App, opts Options) (*Result, error) {
	// Wire injectable deps. In production both come from App; in tests a
	// fake Executor and any Stderr is supplied directly, so `a` may be nil.
	exec := opts.Exec
	if exec == nil {
		exec = NewRealExecutor(a)
	}
	progress := opts.Stderr
	if progress == nil && a != nil {
		progress = a.IO.Err
	}
	if progress == nil {
		progress = io.Discard
	}
	runStdout, runStderr := io.Discard, io.Discard
	if a != nil {
		runStdout, runStderr = a.IO.Out, a.IO.Err
	}

	if len(opts.Instructions) == 0 {
		return nil, fmt.Errorf("empty Dockerfile: no instructions")
	}

	// v1: reject multi-stage.
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

	// ---- FROM --------------------------------------------------------------
	var vmID, baseCommit string
	var err error
	vmID, baseCommit, err = doFrom(ctx, exec, first.From, opts)
	if err != nil {
		return nil, err
	}
	// Ensure teardown on mid-build failure.
	defer func() {
		if err != nil && !opts.Keep && vmID != "" {
			// Best-effort. Use a fresh background ctx so a cancelled parent
			// doesn't prevent cleanup.
			_ = exec.DeleteVM(context.Background(), vmID)
		}
	}()

	fmt.Fprintf(progress, "Step 1/%d : %s\n", len(opts.Instructions), first.Raw)
	if baseCommit != "" {
		fmt.Fprintf(progress, " ---> using base commit %s\n", short(baseCommit))
	} else {
		fmt.Fprintf(progress, " ---> built fresh VM %s\n", short(vmID))
	}

	parentCommit := baseCommit
	res := &Result{StepCount: len(opts.Instructions)}

	// ---- Loop over remaining instructions ---------------------------------
	for i := 1; i < len(opts.Instructions); i++ {
		ins := opts.Instructions[i]
		fmt.Fprintf(progress, "Step %d/%d : %s\n", i+1, len(opts.Instructions), ins.Raw)

		if handled, metaErr := applyMeta(st, ins); handled {
			if metaErr != nil {
				err = metaErr
				return nil, err
			}
			fmt.Fprintf(progress, " ---> metadata\n")
			continue
		}

		key, _, kerr := cacheKeyFor(ins, st, opts.Context, parentCommit)
		if kerr != nil {
			err = kerr
			return nil, err
		}

		if !opts.NoCache {
			if cached := cache.Get(key); cached != "" {
				newVM, switchErr := switchToCommit(ctx, exec, vmID, cached)
				if switchErr != nil {
					fmt.Fprintf(progress, " ---> cache entry stale (%v), rebuilding\n", switchErr)
				} else {
					vmID = newVM
					parentCommit = cached
					res.CachedCount++
					fmt.Fprintf(progress, " ---> using cached layer %s\n", short(cached))
					continue
				}
			}
		}

		if execErr := runStep(ctx, exec, vmID, st, ins, opts.Context, runStdout, runStderr); execErr != nil {
			err = fmt.Errorf("step %d (%s): %w", i+1, ins.Kind, execErr)
			return nil, err
		}

		commitID, cerr := exec.Commit(ctx, vmID)
		if cerr != nil {
			err = fmt.Errorf("commit after step %d: %w", i+1, cerr)
			return nil, err
		}
		parentCommit = commitID
		cache.Put(key, parentCommit)
		cache.Save()
		fmt.Fprintf(progress, " ---> %s\n", short(parentCommit))
	}

	if parentCommit == "" {
		// Only FROM was present — still produce a commit so the build has an output.
		commitID, cerr := exec.Commit(ctx, vmID)
		if cerr != nil {
			err = fmt.Errorf("commit initial state: %w", cerr)
			return nil, err
		}
		parentCommit = commitID
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

	if opts.Tag != "" {
		if terr := exec.CreateTag(ctx, opts.Tag, parentCommit); terr != nil {
			fmt.Fprintf(progress, "warning: failed to create tag %q: %v\n", opts.Tag, terr)
		} else {
			res.Tag = opts.Tag
		}
	}

	if opts.Keep {
		res.BuilderVmID = vmID
	} else {
		if derr := exec.DeleteVM(context.Background(), vmID); derr != nil {
			fmt.Fprintf(progress, "warning: failed to delete builder VM %s: %v\n", vmID, derr)
		}
	}

	return res, nil
}

// doFrom materializes the base VM. Returns (vmID, baseCommitID).
// baseCommitID is "" when starting from scratch.
func doFrom(ctx context.Context, exec Executor, f *dockerfile.FromInstr, opts Options) (string, string, error) {
	if f.As != "" {
		return "", "", fmt.Errorf("FROM ... AS is not supported in single-stage v1")
	}
	if strings.EqualFold(f.Ref, "scratch") {
		if opts.MemSizeMib == 0 || opts.VcpuCount == 0 || opts.FsSizeVmMib == 0 {
			return "", "", fmt.Errorf("FROM scratch requires --mem-size, --vcpu-count, and --fs-size-vm-mib")
		}
		vmID, err := exec.NewVM(ctx, VMSpec{
			MemSizeMib:  opts.MemSizeMib,
			VcpuCount:   opts.VcpuCount,
			FsSizeVmMib: opts.FsSizeVmMib,
			RootfsName:  opts.RootfsName,
			KernelName:  opts.KernelName,
		})
		if err != nil {
			return "", "", fmt.Errorf("FROM scratch: %w", err)
		}
		return vmID, "", nil
	}

	commitID := f.Ref
	if resolved, ok := exec.ResolveTag(ctx, f.Ref); ok {
		commitID = resolved
	}
	vmID, err := exec.RestoreFromCommit(ctx, commitID)
	if err != nil {
		return "", "", fmt.Errorf("FROM %s: %w", f.Ref, err)
	}
	return vmID, commitID, nil
}

// switchToCommit restores a fresh VM from the cached commit and deletes
// the old one. Returns the new VM id.
func switchToCommit(ctx context.Context, exec Executor, oldVM, commitID string) (string, error) {
	newVM, err := exec.RestoreFromCommit(ctx, commitID)
	if err != nil {
		return "", err
	}
	if oldVM != "" {
		_ = exec.DeleteVM(ctx, oldVM)
	}
	return newVM, nil
}

// applyMeta handles instructions that only update builder state, with no
// corresponding VM mutation.
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

// cacheKeyFor produces a cache key for a materializing instruction
// (RUN / COPY / ADD). Metadata instructions don't create layers.
func cacheKeyFor(ins dockerfile.Instruction, st *state, bc *BuildContext, parentCommit string) (string, []string, error) {
	extras := metaExtras(st)
	switch ins.Kind {
	case dockerfile.KindRun:
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

// metaExtras produces a deterministic slice encoding the builder's env /
// workdir / user so future layer keys depend on them.
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

// runStep dispatches RUN and COPY/ADD execution against the live VM.
func runStep(ctx context.Context, exec Executor, vmID string, st *state, ins dockerfile.Instruction, bc *BuildContext, stdout, stderr io.Writer) error {
	switch ins.Kind {
	case dockerfile.KindRun:
		return doRun(ctx, exec, vmID, st, ins.Run, stdout, stderr)
	case dockerfile.KindCopy, dockerfile.KindAdd:
		return doCopy(ctx, exec, vmID, st, ins.Copy, bc, stdout, stderr)
	}
	return fmt.Errorf("runStep: unhandled instruction %s", ins.Kind)
}

func doRun(ctx context.Context, exec Executor, vmID string, st *state, r *dockerfile.RunInstr, stdout, stderr io.Writer) error {
	cmd := runCommand(r, st)
	code, err := exec.Run(ctx, vmID, cmd, copyMap(st.env), st.workdir, stdout, stderr)
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

// runCommandLine is a flat, stable representation of the command for cache
// keys only.
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

func doCopy(ctx context.Context, exec Executor, vmID string, st *state, c *dockerfile.CopyInstr, bc *BuildContext, stdout, stderr io.Writer) error {
	if c.From != "" {
		return fmt.Errorf("COPY --from is not supported in v1")
	}
	dest := c.Dest
	if !filepath.IsAbs(dest) {
		base := st.workdir
		if base == "" {
			base = "/"
		}
		dest = filepath.Join(base, dest)
	}
	destIsDir := strings.HasSuffix(c.Dest, "/") || len(c.Sources) > 1

	mkdirTarget := dest
	if !destIsDir {
		mkdirTarget = filepath.Dir(dest)
	}
	if mkdirTarget != "" && mkdirTarget != "/" {
		if err := execSimple(ctx, exec, vmID, []string{"mkdir", "-p", mkdirTarget}, stdout, stderr); err != nil {
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
		if err := exec.Upload(ctx, vmID, srcAbs, remoteTarget, st2.IsDir()); err != nil {
			return fmt.Errorf("upload %s: %w", srcSpec, err)
		}
	}

	if c.Chown != "" {
		if err := execSimple(ctx, exec, vmID, []string{"chown", "-R", c.Chown, dest}, stdout, stderr); err != nil {
			return fmt.Errorf("chown: %w", err)
		}
	}
	return nil
}

// execSimple runs a management command, discarding output but respecting
// exit code. Used for mkdir / chown inside COPY.
func execSimple(ctx context.Context, exec Executor, vmID string, cmd []string, stdout, stderr io.Writer) error {
	code, err := exec.Run(ctx, vmID, cmd, nil, "", io.Discard, io.Discard)
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

// expand performs lightweight $VAR / ${VAR} substitution across ENV + build args.
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

// utilsShellJoin locally quotes argv elements (so we don't pull in utils
// just for shell joining inside this package).
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
