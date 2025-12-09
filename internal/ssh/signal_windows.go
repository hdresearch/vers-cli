//go:build windows

package ssh

import (
	"os"
)

// sigWinch returns an empty slice on Windows as SIGWINCH doesn't exist.
// Terminal resize is handled differently on Windows.
func sigWinch() []os.Signal {
	return []os.Signal{}
}
