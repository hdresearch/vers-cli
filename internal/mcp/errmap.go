package mcp

import (
	"errors"
	"strings"

	"github.com/hdresearch/vers-cli/internal/errorsx"
)

// mapMCPError converts internal errors to stable, MCP-facing coded errors.
func mapMCPError(err error) error {
	if err == nil {
		return nil
	}

	var hasChildren *errorsx.HasChildrenError
	if errors.As(err, &hasChildren) {
		return Err(E_CONFLICT, "VM has children; set recursive=true to delete all descendants", map[string]any{"vmId": hasChildren.VMID})
	}
	var isRoot *errorsx.IsRootError
	if errors.As(err, &isRoot) {
		return Err(E_CONFLICT, "Cannot delete root VM; branch from it first to preserve the topology", map[string]any{"vmId": isRoot.VMID})
	}
	// Heuristics for common cases
	if strings.Contains(strings.ToLower(err.Error()), "not found") {
		return Err(E_NOT_FOUND, err.Error(), nil)
	}
	return err
}
