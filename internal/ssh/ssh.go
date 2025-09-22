package ssh

import (
	"fmt"
	"os/exec"
	"strconv"
)

// getTimeout returns the SSH/SCP connect timeout in seconds.
// Currently fixed at 5s to match CLI behavior; can be extended to read env/flags.
func getTimeout() string { return "5" }

// SSHCommand builds an ssh command with consistent options.
// extraArgs may include a remote command string.
func SSHCommand(host, port, keyPath string, extraArgs ...string) *exec.Cmd {
	args := []string{
		fmt.Sprintf("root@%s", host),
		"-p", port,
		"-o", "StrictHostKeyChecking=no",
		"-o", "UserKnownHostsFile=/dev/null",
		"-o", "IdentitiesOnly=yes",
		"-o", "PreferredAuthentications=publickey",
		"-o", "LogLevel=ERROR",
		"-o", "ConnectTimeout=" + getTimeout(),
		"-i", keyPath,
	}
	args = append(args, extraArgs...)
	return exec.Command("ssh", args...)
}

// SSHArgs builds argument list for ssh with consistent options.
// extraArgs may include a remote command string.
func SSHArgs(host, port, keyPath string, extraArgs ...string) []string {
	args := []string{
		fmt.Sprintf("root@%s", host),
		"-p", port,
		"-o", "StrictHostKeyChecking=no",
		"-o", "UserKnownHostsFile=/dev/null",
		"-o", "IdentitiesOnly=yes",
		"-o", "PreferredAuthentications=publickey",
		"-o", "LogLevel=ERROR",
		"-o", "ConnectTimeout=" + getTimeout(),
		"-i", keyPath,
	}
	args = append(args, extraArgs...)
	return args
}

// SCPArgs builds argument list for scp with consistent options.
// Set recursive to add -r. The source and dest are appended by caller.
func SCPArgs(port, keyPath string, recursive bool) []string {
	args := []string{
		"-P", port,
		"-o", "StrictHostKeyChecking=no",
		"-o", "UserKnownHostsFile=/dev/null",
		"-o", "IdentitiesOnly=yes",
		"-o", "PreferredAuthentications=publickey",
		"-o", "LogLevel=ERROR",
		"-o", "ConnectTimeout=" + getTimeout(),
		"-i", keyPath,
	}
	if recursive {
		args = append(args, "-r")
	}
	return args
}

// PortToString ensures port is a string for scp/ssh flags.
func PortToString(port int) string { return strconv.Itoa(port) }
