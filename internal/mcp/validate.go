package mcp

import "fmt"

func validateRun(in RunInput) error {
	if in.MemSizeMib < 0 || in.VcpuCount < 0 || in.FsSizeVmMib < 0 {
		return fmt.Errorf("invalid sizes: negative values are not allowed")
	}
	return nil
}

func validateExecute(in ExecuteInput) error {
	if len(in.Command) == 0 {
		return Err(E_INVALID, "command is required", nil)
	}
	if in.TimeoutSeconds < 0 {
		return Err(E_INVALID, "timeoutSeconds must be >= 0", nil)
	}
	return nil
}

func validateBranch(in BranchInput) error {
	if in.Count < 0 {
		return Err(E_INVALID, "count must be >= 1", nil)
	}
	if in.Count > 1 {
		if in.Alias != "" {
			return Err(E_INVALID, "alias cannot be used when creating multiple branches", nil)
		}
	}
	return nil
}

func validateKill(in KillInput) error {
	if !in.SkipConfirmation {
		return Err(E_CONFIRM_REQUIRED, "skipConfirmation=true required for destructive operations in MCP", map[string]any{"hint": "Set skipConfirmation to true to proceed non-interactively."})
	}
	return nil
}
