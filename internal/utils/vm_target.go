package utils

import (
	"regexp"
)

// uuidRegex matches a standard UUID v4 format.
var uuidRegex = regexp.MustCompile(`^[0-9a-fA-F]{8}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{12}$`)

// LooksLikeVMTarget returns true if the string looks like a VM identifier
// (a UUID or a known alias), as opposed to a shell command.
func LooksLikeVMTarget(s string) bool {
	if uuidRegex.MatchString(s) {
		return true
	}

	// Check if it's a known alias
	resolved := ResolveAlias(s)
	if resolved != s {
		return true
	}

	return false
}
