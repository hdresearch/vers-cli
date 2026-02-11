package utils

import "strings"

// ShellQuote returns s quoted so it is treated as a single token by a
// POSIX shell. If s is already "safe" (only alphanumerics, hyphens,
// underscores, dots, forward-slashes, colons, equals, and commas)
// it is returned unchanged. Otherwise it is wrapped in single quotes
// with any embedded single quotes escaped.
func ShellQuote(s string) string {
	if s == "" {
		return "''"
	}
	safe := true
	for _, c := range s {
		if !isSafeShellChar(c) {
			safe = false
			break
		}
	}
	if safe {
		return s
	}
	return "'" + strings.ReplaceAll(s, "'", `'\''`) + "'"
}

// ShellJoin quotes each argument and joins them with spaces, producing
// a command string safe for passing to a remote shell.
func ShellJoin(args []string) string {
	quoted := make([]string, len(args))
	for i, a := range args {
		quoted[i] = ShellQuote(a)
	}
	return strings.Join(quoted, " ")
}

func isSafeShellChar(c rune) bool {
	return (c >= 'a' && c <= 'z') ||
		(c >= 'A' && c <= 'Z') ||
		(c >= '0' && c <= '9') ||
		c == '-' || c == '_' || c == '.' ||
		c == '/' || c == ':' || c == '=' ||
		c == ','
}
