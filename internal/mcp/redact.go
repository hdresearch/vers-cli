package mcp

import (
	"os"
	"strings"
)

// redact masks known sensitive values from log and summary strings.
func redact(s string) string {
	if s == "" {
		return s
	}
	if v := os.Getenv("VERS_API_KEY"); v != "" {
		s = strings.ReplaceAll(s, v, "***")
	}
	return s
}
