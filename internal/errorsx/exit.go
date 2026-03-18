package errorsx

import "strings"

// Exit codes for structured error handling in scripts and agents.
const (
	ExitOK         = 0
	ExitGeneral    = 1 // generic / unknown error
	ExitAuth       = 2 // authentication failure (401/403)
	ExitNotFound   = 3 // resource not found (404)
	ExitConflict   = 4 // conflict (409)
	ExitBadRequest = 5 // invalid input / bad request (400)
	ExitTimeout    = 6 // operation timed out
	ExitCancelled  = 7 // user cancelled (e.g. declined confirmation)
)

// ExitCodeFromError returns an appropriate exit code for the given error.
func ExitCodeFromError(err error) int {
	if err == nil {
		return ExitOK
	}
	s := err.Error()
	lower := strings.ToLower(s)

	switch {
	case strings.Contains(s, "401") || strings.Contains(lower, "unauthorized") ||
		strings.Contains(s, "403") || strings.Contains(lower, "forbidden"):
		return ExitAuth
	case strings.Contains(lower, "not found") || strings.Contains(s, "404"):
		return ExitNotFound
	case strings.Contains(s, "409"):
		return ExitConflict
	case strings.Contains(s, "400") || strings.Contains(lower, "bad request"):
		return ExitBadRequest
	case strings.Contains(lower, "timed out") || strings.Contains(lower, "timeout") ||
		strings.Contains(lower, "deadline exceeded"):
		return ExitTimeout
	case strings.Contains(lower, "cancelled by user") || strings.Contains(lower, "operation cancelled"):
		return ExitCancelled
	default:
		return ExitGeneral
	}
}
