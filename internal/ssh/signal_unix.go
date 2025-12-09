//go:build !windows

package ssh

import (
	"os"
	"syscall"
)

// sigWinch returns the signals to watch for terminal resize on Unix systems.
func sigWinch() []os.Signal {
	return []os.Signal{syscall.SIGWINCH}
}
