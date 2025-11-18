package ssh

import (
	"strings"
	"testing"
)

func TestSSHCommand_BuildsExpectedArgs(t *testing.T) {
	cmd := SSHCommand("1.2.3.4", "2222", "/path/to/key", "echo hi")
	args := cmd.Args
	joined := strings.Join(args, " ")
	// Binary and target
	if !strings.Contains(joined, "ssh ") || !strings.Contains(joined, "root@1.2.3.4") {
		t.Fatalf("unexpected ssh args: %v", args)
	}
	// Key, timeout, ProxyCommand for SSH-over-TLS
	mustContain(t, joined, "-i /path/to/key")
	mustContain(t, joined, "ConnectTimeout=")
	mustContain(t, joined, "ProxyCommand=")
	// Command propagated
	mustContain(t, joined, "echo hi")
}

func TestSCPArgs_IncludesRecursiveAndTimeout(t *testing.T) {
	args := SCPArgs("2222", "/k", true)
	joined := strings.Join(args, " ")
	mustContain(t, joined, "-P 2222")
	mustContain(t, joined, "-i /k")
	mustContain(t, joined, "-r")
	mustContain(t, joined, "ConnectTimeout=")
}

func mustContain(t *testing.T, s, sub string) {
	t.Helper()
	if !strings.Contains(s, sub) {
		t.Fatalf("expected %q to contain %q", s, sub)
	}
}
