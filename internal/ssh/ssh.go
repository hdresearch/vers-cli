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
	// Use SSH-over-TLS via proxy (bypass load balancer which terminates TLS)
	proxyHost := "44.210.239.66" // Direct connection to proxy server
	vmHostname := fmt.Sprintf("%s.vm.vers.sh", host) // SNI hostname
	proxyCommand := fmt.Sprintf("openssl s_client -connect %s:443 -servername %s -quiet 2>/dev/null", proxyHost, vmHostname)

	args := []string{
		fmt.Sprintf("root@%s", vmHostname),
		"-o", fmt.Sprintf("ProxyCommand=%s", proxyCommand),
		"-o", "StrictHostKeyChecking=no",
		"-o", "UserKnownHostsFile=/dev/null",
		"-o", "IdentitiesOnly=yes",
		"-o", "PreferredAuthentications=publickey",
		"-o", "LogLevel=ERROR",
		"-o", "ConnectTimeout=30",
		"-o", "ServerAliveInterval=10",
		"-o", "ServerAliveCountMax=6",
		"-o", "TCPKeepAlive=yes",
		"-i", keyPath,
	}
	args = append(args, extraArgs...)
	return exec.Command("ssh", args...)
}

// SSHArgs builds argument list for ssh with consistent options.
// extraArgs may include a remote command string.
func SSHArgs(host, port, keyPath string, extraArgs ...string) []string {
	// Use SSH-over-TLS via proxy (bypass load balancer which terminates TLS)
	proxyHost := "44.210.239.66" // Direct connection to proxy server
	vmHostname := fmt.Sprintf("%s.vm.vers.sh", host) // SNI hostname
	proxyCommand := fmt.Sprintf("openssl s_client -connect %s:443 -servername %s -quiet 2>/dev/null", proxyHost, vmHostname)

	args := []string{
		fmt.Sprintf("root@%s", vmHostname),
		"-o", fmt.Sprintf("ProxyCommand=%s", proxyCommand),
		"-o", "StrictHostKeyChecking=no",
		"-o", "UserKnownHostsFile=/dev/null",
		"-o", "IdentitiesOnly=yes",
		"-o", "PreferredAuthentications=publickey",
		"-o", "LogLevel=ERROR",
		"-o", "ConnectTimeout=30",
		"-o", "ServerAliveInterval=10",
		"-o", "ServerAliveCountMax=6",
		"-o", "TCPKeepAlive=yes",
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
