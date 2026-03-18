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
func ParseFormat(quiet bool, formatStr string) OutputFormat {
	if quiet {
		return FormatQuiet
	}
	if formatStr == "json" {
		return FormatJSON
	}
	return FormatDefault
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
