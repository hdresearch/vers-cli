package cmd

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"syscall"

	"github.com/hdresearch/vers-cli/internal/handlers"
	"github.com/spf13/cobra"
)

var tunnelCmd = &cobra.Command{
	Use:   "tunnel [vm-id|alias] <local-port>:<remote-port> | <local-port>:<remote-host>:<remote-port>",
	Short: "Forward a local port to a VM",
	Long: `Create an SSH tunnel that forwards a local port to a port on the VM.
If no VM ID or alias is provided, uses the current HEAD.

This is equivalent to ssh -L. Traffic to 127.0.0.1:<local-port> on your
machine is forwarded through the SSH connection to <remote-host>:<remote-port>
on the VM. If <remote-host> is omitted, it defaults to localhost.

Examples:
  vers tunnel 8080:80            Forward local 8080 to port 80 on HEAD VM
  vers tunnel 3000:3000          Forward local 3000 to port 3000 on HEAD VM
  vers tunnel my-vm 5432:5432    Forward local 5432 to port 5432 on my-vm
  vers tunnel 9090:10.0.0.2:80  Forward local 9090 to 10.0.0.2:80 via the VM`,
	Args: cobra.RangeArgs(1, 2),
	RunE: func(cmd *cobra.Command, args []string) error {
		// Parse args: either [spec] or [target, spec]
		var target, spec string
		if len(args) == 1 {
			spec = args[0]
		} else {
			target = args[0]
			spec = args[1]
		}

		localPort, remoteHost, remotePort, err := parseTunnelSpec(spec)
		if err != nil {
			return err
		}

		// Use a cancellable context that responds to Ctrl-C
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		sigCh := make(chan os.Signal, 1)
		signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
		go func() {
			<-sigCh
			cancel()
		}()

		_, err = handlers.HandleTunnel(ctx, application, handlers.TunnelReq{
			Target:     target,
			LocalPort:  localPort,
			RemoteHost: remoteHost,
			RemotePort: remotePort,
		})
		return err
	},
}

// parseTunnelSpec parses "localPort:remotePort" or "localPort:remoteHost:remotePort".
func parseTunnelSpec(spec string) (localPort int, remoteHost string, remotePort int, err error) {
	parts := strings.Split(spec, ":")
	switch len(parts) {
	case 2:
		// localPort:remotePort
		remoteHost = "localhost"
		localPort, err = strconv.Atoi(parts[0])
		if err != nil {
			return 0, "", 0, fmt.Errorf("invalid local port %q: %w", parts[0], err)
		}
		remotePort, err = strconv.Atoi(parts[1])
		if err != nil {
			return 0, "", 0, fmt.Errorf("invalid remote port %q: %w", parts[1], err)
		}
	case 3:
		// localPort:remoteHost:remotePort
		localPort, err = strconv.Atoi(parts[0])
		if err != nil {
			return 0, "", 0, fmt.Errorf("invalid local port %q: %w", parts[0], err)
		}
		remoteHost = parts[1]
		remotePort, err = strconv.Atoi(parts[2])
		if err != nil {
			return 0, "", 0, fmt.Errorf("invalid remote port %q: %w", parts[2], err)
		}
	default:
		return 0, "", 0, fmt.Errorf("invalid tunnel spec %q: expected <local-port>:<remote-port> or <local-port>:<remote-host>:<remote-port>", spec)
	}

	if localPort < 0 || localPort > 65535 {
		return 0, "", 0, fmt.Errorf("local port %d out of range (0-65535)", localPort)
	}
	if remotePort < 1 || remotePort > 65535 {
		return 0, "", 0, fmt.Errorf("remote port %d out of range (1-65535)", remotePort)
	}

	return localPort, remoteHost, remotePort, nil
}

func init() {
	rootCmd.AddCommand(tunnelCmd)
}
