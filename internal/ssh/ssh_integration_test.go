package ssh

import (
    "os"
    "os/exec"
    "path/filepath"
    "runtime"
    "testing"
)

// createStub writes a small shell script with the given name that captures args to outPath.
func createStub(t *testing.T, dir, name, outPath string) string {
    t.Helper()
    path := filepath.Join(dir, name)
    // On Windows, bash may not be available; skip on windows.
    if runtime.GOOS == "windows" {
        t.Skip("ssh integration stub not supported on windows")
    }
    script := "#!/usr/bin/env bash\n" +
        "printf '%s ' \"$@\" > \"" + outPath + "\"\n"
    if err := os.WriteFile(path, []byte(script), 0o755); err != nil {
        t.Fatalf("failed to write stub %s: %v", name, err)
    }
    return path
}

func withPathPrepended(t *testing.T, newDir string) func() {
    t.Helper()
    old := os.Getenv("PATH")
    _ = os.Setenv("PATH", newDir+string(os.PathListSeparator)+old)
    return func() { _ = os.Setenv("PATH", old) }
}

func TestSSHCommand_ExecutesStubWithExpectedArgs(t *testing.T) {
    tmp := t.TempDir()
    out := filepath.Join(tmp, "ssh_args.txt")
    _ = createStub(t, tmp, "ssh", out)
    restore := withPathPrepended(t, tmp)
    defer restore()

    cmd := SSHCommand("10.0.0.1", "2200", "/tmp/key", "echo", "hello")
    // Use current env so PATH override applies
    cmd.Env = os.Environ()
    if err := cmd.Run(); err != nil {
        t.Fatalf("ssh stub did not run: %v", err)
    }
    data, err := os.ReadFile(out)
    if err != nil {
        t.Fatalf("failed to read stub output: %v", err)
    }
    got := string(data)
    // Check core flags/args present
    mustContain(t, got, "root@10.0.0.1")
    mustContain(t, got, "-p 2200")
    mustContain(t, got, "-i /tmp/key")
    mustContain(t, got, "ConnectTimeout=")
    // Command propagated
    mustContain(t, got, "echo")
    mustContain(t, got, "hello")
}

func TestSCPArgs_ExecutesStubWithExpectedArgs(t *testing.T) {
    tmp := t.TempDir()
    out := filepath.Join(tmp, "scp_args.txt")
    _ = createStub(t, tmp, "scp", out)
    restore := withPathPrepended(t, tmp)
    defer restore()

    args := SCPArgs("2200", "/tmp/key", true)
    // add source/dest to complete the command
    args = append(args, "/local/file.txt", "root@host:/remote/file.txt")
    cmd := execCommand("scp", args...)
    cmd.Env = os.Environ()
    if err := cmd.Run(); err != nil {
        t.Fatalf("scp stub did not run: %v", err)
    }
    data, err := os.ReadFile(out)
    if err != nil {
        t.Fatalf("failed to read stub output: %v", err)
    }
    got := string(data)
    mustContain(t, got, "-P 2200")
    mustContain(t, got, "-i /tmp/key")
    mustContain(t, got, "-r")
    mustContain(t, got, "ConnectTimeout=")
    mustContain(t, got, "/local/file.txt")
    mustContain(t, got, "root@host:/remote/file.txt")
}

// execCommand wraps os/exec.Command to avoid import cycles in test; declare here.
func execCommand(name string, arg ...string) *exec.Cmd { return exec.Command(name, arg...) }

