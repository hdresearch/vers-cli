package presenters

import (
	"encoding/json"
	"fmt"
	"os"
)

// OutputFormat controls how command output is rendered.
type OutputFormat int

const (
	FormatDefault OutputFormat = iota
	FormatQuiet                // just IDs/names, one per line
	FormatJSON                 // full JSON
)

// ParseFormat returns the output format from flag values.
//
// Precedence: quiet > json > format > default. The legacy --format flag is
// accepted for backwards compatibility but only "json" is valid; any other
// value returns an enumerated error so agents can self-correct in one retry.
//
// Callers should pass the values of --quiet, --json, and --format directly;
// deprecation of --format is handled by cobra's MarkDeprecated at the flag
// registration site, which prints a warning to stderr when --format is used.
func ParseFormat(quiet bool, jsonFlag bool, formatStr string) (OutputFormat, error) {
	if quiet {
		return FormatQuiet, nil
	}
	if formatStr != "" {
		if formatStr != "json" {
			return FormatDefault, fmt.Errorf(`--format must be "json" (got: %q). Note: --format is deprecated, use --json instead`, formatStr)
		}
		return FormatJSON, nil
	}
	if jsonFlag {
		return FormatJSON, nil
	}
	return FormatDefault, nil
}

// PrintQuiet prints each string on its own line to stdout.
func PrintQuiet(items []string) {
	for _, item := range items {
		fmt.Println(item)
	}
}

// PrintJSON marshals v to indented JSON and prints to stdout.
func PrintJSON(v interface{}) error {
	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	return enc.Encode(v)
}
