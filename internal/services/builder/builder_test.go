package builder

import (
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/hdresearch/vers-cli/internal/dockerfile"
)

// newState constructs an empty builder state for tests.
func newState() *state {
	return &state{
		env:          map[string]string{},
		argVals:      map[string]string{},
		declaredArgs: map[string]bool{},
		labels:       map[string]string{},
	}
}

// --- expand -----------------------------------------------------------------

func TestExpand(t *testing.T) {
	st := newState()
	st.env["FOO"] = "bar"
	st.env["NAME"] = "app"
	st.declaredArgs["VERSION"] = true
	st.argVals["VERSION"] = "1.2.3"

	cases := map[string]string{
		"$FOO":                       "bar",
		"${FOO}":                     "bar",
		"$VERSION":                   "1.2.3",
		"${VERSION}-${NAME}":         "1.2.3-app",
		"literal no vars":            "literal no vars",
		"$UNSET":                     "",
		"prefix-$FOO-suffix":         "prefix-bar-suffix",
		"${NAME}_${VERSION}":         "app_1.2.3",
		"$":                          "$",     // lone $
		"${unterminated":             "${unterminated",
		"$ foo":                      "$ foo", // $ with no identifier following
		"trail-$NAME.txt":            "trail-app.txt",
	}
	for in, want := range cases {
		if got := expand(in, st); got != want {
			t.Errorf("expand(%q): got %q, want %q", in, got, want)
		}
	}
}

func TestExpand_UndeclaredArgNotUsed(t *testing.T) {
	// Values in argVals are only applied if the ARG was declared in the
	// Dockerfile (ARG NAME). Build-args passed in without a matching ARG
	// should be ignored to match Docker's semantics.
	st := newState()
	st.argVals["SECRET"] = "nope"
	if got := expand("$SECRET", st); got != "" {
		t.Errorf("expected empty (undeclared ARG), got %q", got)
	}
}

// --- applyMeta --------------------------------------------------------------

func TestApplyMeta_EnvExpandsPriorVars(t *testing.T) {
	st := newState()
	st.env["ROOT"] = "/srv"
	ins := dockerfile.Instruction{
		Kind: dockerfile.KindEnv,
		Env:  &dockerfile.KVInstr{Pairs: []dockerfile.KV{{Key: "APP_DIR", Value: "${ROOT}/app"}}},
	}
	handled, err := applyMeta(st, ins)
	if !handled || err != nil {
		t.Fatalf("handled=%v err=%v", handled, err)
	}
	if st.env["APP_DIR"] != "/srv/app" {
		t.Errorf("got %q", st.env["APP_DIR"])
	}
}

func TestApplyMeta_Workdir(t *testing.T) {
	st := newState()
	// Absolute path sets directly.
	_, _ = applyMeta(st, dockerfile.Instruction{Kind: dockerfile.KindWorkdir, Workdir: &dockerfile.StrInstr{Value: "/app"}})
	if st.workdir != "/app" {
		t.Errorf("workdir=%q", st.workdir)
	}
	// Relative WORKDIR is joined with previous.
	_, _ = applyMeta(st, dockerfile.Instruction{Kind: dockerfile.KindWorkdir, Workdir: &dockerfile.StrInstr{Value: "sub"}})
	if st.workdir != "/app/sub" {
		t.Errorf("workdir=%q", st.workdir)
	}
}

func TestApplyMeta_ArgRespectsPreset(t *testing.T) {
	st := newState()
	st.argVals["V"] = "from-cli"
	_, _ = applyMeta(st, dockerfile.Instruction{Kind: dockerfile.KindArg, Arg: &dockerfile.ArgInstr{Name: "V", Default: "from-df", HasDef: true}})
	if st.argVals["V"] != "from-cli" {
		t.Errorf("expected CLI value to win, got %q", st.argVals["V"])
	}
	if !st.declaredArgs["V"] {
		t.Error("expected ARG to be marked declared")
	}
}

func TestApplyMeta_CmdAndEntrypoint(t *testing.T) {
	st := newState()
	_, _ = applyMeta(st, dockerfile.Instruction{Kind: dockerfile.KindCmd, Cmd: &dockerfile.ExecInstr{Exec: []string{"node", "server.js"}}})
	if !reflect.DeepEqual(st.cmd, []string{"node", "server.js"}) {
		t.Errorf("cmd=%+v", st.cmd)
	}
	_, _ = applyMeta(st, dockerfile.Instruction{Kind: dockerfile.KindEntrypoint, Entrypoint: &dockerfile.ExecInstr{Shell: "/entry.sh"}})
	if st.entrypointShell != "/entry.sh" {
		t.Errorf("entrypointShell=%q", st.entrypointShell)
	}
}

func TestApplyMeta_ExposeAccumulates(t *testing.T) {
	st := newState()
	_, _ = applyMeta(st, dockerfile.Instruction{Kind: dockerfile.KindExpose, Expose: &dockerfile.ExposeInstr{Ports: []string{"80", "443"}}})
	_, _ = applyMeta(st, dockerfile.Instruction{Kind: dockerfile.KindExpose, Expose: &dockerfile.ExposeInstr{Ports: []string{"8080/tcp"}}})
	if !reflect.DeepEqual(st.exposed, []string{"80", "443", "8080/tcp"}) {
		t.Errorf("exposed=%+v", st.exposed)
	}
}

func TestApplyMeta_NotHandledForRun(t *testing.T) {
	st := newState()
	handled, _ := applyMeta(st, dockerfile.Instruction{Kind: dockerfile.KindRun, Run: &dockerfile.RunInstr{Shell: "true"}})
	if handled {
		t.Error("RUN must not be treated as metadata")
	}
}

// --- runCommand / maybeWrapUser --------------------------------------------

func TestRunCommand_ShellForm(t *testing.T) {
	st := newState()
	got := runCommand(&dockerfile.RunInstr{Shell: "echo hi"}, st)
	if !reflect.DeepEqual(got, []string{"bash", "-c", "echo hi"}) {
		t.Errorf("got %+v", got)
	}
}

func TestRunCommand_ExecForm(t *testing.T) {
	st := newState()
	got := runCommand(&dockerfile.RunInstr{Exec: []string{"echo", "hi"}}, st)
	if !reflect.DeepEqual(got, []string{"echo", "hi"}) {
		t.Errorf("got %+v", got)
	}
}

func TestRunCommand_WrapsUser(t *testing.T) {
	st := newState()
	st.user = "node"
	got := runCommand(&dockerfile.RunInstr{Shell: "whoami"}, st)
	// Expect: sudo -u node bash -c "<quoted bash -c whoami>"
	if len(got) < 5 || got[0] != "sudo" || got[1] != "-u" || got[2] != "node" || got[3] != "bash" || got[4] != "-c" {
		t.Errorf("expected sudo wrap, got %+v", got)
	}
	inner := got[len(got)-1]
	if !strings.Contains(inner, "whoami") {
		t.Errorf("inner missing whoami: %q", inner)
	}
}

func TestRunCommand_ExecFormWrapsUser(t *testing.T) {
	st := newState()
	st.user = "1000:1000"
	got := runCommand(&dockerfile.RunInstr{Exec: []string{"echo", "hello world"}}, st)
	if got[0] != "sudo" || got[2] != "1000:1000" {
		t.Errorf("got %+v", got)
	}
	// The shell-joined inner should quote "hello world" so it survives bash -c.
	inner := got[len(got)-1]
	if !strings.Contains(inner, "echo") || !strings.Contains(inner, "hello world") {
		t.Errorf("inner missing quoted args: %q", inner)
	}
}

// --- cacheKeyFor ------------------------------------------------------------

func TestCacheKeyFor_RunChangesWithEnv(t *testing.T) {
	ins := dockerfile.Instruction{Kind: dockerfile.KindRun, Run: &dockerfile.RunInstr{Shell: "echo hi"}, Raw: "RUN echo hi"}
	st1 := newState()
	k1, _, err := cacheKeyFor(ins, st1, nil, "parent")
	if err != nil {
		t.Fatal(err)
	}
	st2 := newState()
	st2.env["FOO"] = "bar"
	k2, _, err := cacheKeyFor(ins, st2, nil, "parent")
	if err != nil {
		t.Fatal(err)
	}
	if k1 == k2 {
		t.Error("expected ENV delta to change cache key")
	}
}

func TestCacheKeyFor_RunChangesWithWorkdirAndUser(t *testing.T) {
	ins := dockerfile.Instruction{Kind: dockerfile.KindRun, Run: &dockerfile.RunInstr{Shell: "ls"}, Raw: "RUN ls"}
	base := newState()
	k0, _, _ := cacheKeyFor(ins, base, nil, "p")

	wd := newState()
	wd.workdir = "/app"
	kwd, _, _ := cacheKeyFor(ins, wd, nil, "p")

	user := newState()
	user.user = "node"
	ku, _, _ := cacheKeyFor(ins, user, nil, "p")

	if k0 == kwd || k0 == ku || kwd == ku {
		t.Errorf("expected distinct keys, got %s %s %s", k0, kwd, ku)
	}
}

func TestCacheKeyFor_CopyIncludesTreeHash(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, "f.txt"), "one")
	bc, err := LoadContext(dir)
	if err != nil {
		t.Fatal(err)
	}
	ins := dockerfile.Instruction{
		Kind: dockerfile.KindCopy,
		Copy: &dockerfile.CopyInstr{Sources: []string{"f.txt"}, Dest: "/dst"},
		Raw:  "COPY f.txt /dst",
	}
	k1, _, err := cacheKeyFor(ins, newState(), bc, "p")
	if err != nil {
		t.Fatal(err)
	}
	writeFile(t, filepath.Join(dir, "f.txt"), "two")
	k2, _, err := cacheKeyFor(ins, newState(), bc, "p")
	if err != nil {
		t.Fatal(err)
	}
	if k1 == k2 {
		t.Error("expected tree mutation to change cache key")
	}
}

func TestCacheKeyFor_CopyFromRejected(t *testing.T) {
	bc, _ := LoadContext(t.TempDir())
	ins := dockerfile.Instruction{
		Kind: dockerfile.KindCopy,
		Copy: &dockerfile.CopyInstr{Sources: []string{"x"}, Dest: "/dst", From: "builder"},
		Raw:  "COPY --from=builder x /dst",
	}
	if _, _, err := cacheKeyFor(ins, newState(), bc, "p"); err == nil {
		t.Error("expected error for COPY --from")
	}
}

// --- LayerCache -------------------------------------------------------------

func TestLayerCache_RoundTrip(t *testing.T) {
	dir := t.TempDir()
	t.Chdir(dir)

	if err := os.MkdirAll(".vers", 0755); err != nil {
		t.Fatal(err)
	}
	c := LoadCache()
	if len(c.Entries) != 0 {
		t.Errorf("new cache should be empty, got %+v", c.Entries)
	}
	c.Put("k1", "commit-a")
	c.Put("k2", "commit-b")
	c.Save()

	raw, err := os.ReadFile(".vers/buildcache.json")
	if err != nil {
		t.Fatal(err)
	}
	var back LayerCache
	if err := json.Unmarshal(raw, &back); err != nil {
		t.Fatal(err)
	}
	if back.Entries["k1"] != "commit-a" || back.Entries["k2"] != "commit-b" {
		t.Errorf("round-trip mismatch: %+v", back.Entries)
	}

	// Reload via LoadCache()
	c2 := LoadCache()
	if c2.Get("k1") != "commit-a" {
		t.Errorf("LoadCache lost entry: %+v", c2.Entries)
	}
}

func TestLayerCache_NoVersDirIsNoOp(t *testing.T) {
	dir := t.TempDir()
	t.Chdir(dir)
	// No .vers/ — save must silently no-op.
	c := LoadCache()
	c.Put("k", "v")
	c.Save()
	if _, err := os.Stat(".vers/buildcache.json"); !os.IsNotExist(err) {
		t.Errorf("expected no cache file without .vers/, got err=%v", err)
	}
	// In-memory state is still usable.
	if c.Get("k") != "v" {
		t.Error("in-memory Put should still work")
	}
}

func TestLayerCache_NilSafe(t *testing.T) {
	var c *LayerCache
	if c.Get("anything") != "" {
		t.Error("nil.Get should return empty")
	}
	c.Put("a", "b") // must not panic
	c.Save()        // must not panic
}
