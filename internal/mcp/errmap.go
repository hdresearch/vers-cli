package mcp

import (
	"strings"
)

// mapMCPError converts internal errors to stable, MCP-facing coded errors.
func mapMCPError(err error) error {
	if err == nil {
		return nil
	}
	if strings.Contains(strings.ToLower(err.Error()), "not found") {
		return Err(E_NOT_FOUND, err.Error(), nil)
	}
	return err
}
