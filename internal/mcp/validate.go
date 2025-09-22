package mcp

import "fmt"

func validateRun(in RunInput) error {
	if in.MemSizeMib < 0 || in.VcpuCount < 0 || in.FsSizeClusterMib < 0 || in.FsSizeVmMib < 0 {
		return fmt.Errorf("invalid sizes: negative values are not allowed")
	}
	if in.FsSizeVmMib > 0 && in.FsSizeClusterMib > 0 && in.FsSizeVmMib > in.FsSizeClusterMib {
		return fmt.Errorf("vm fs size must not exceed cluster fs size")
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
	// Alias optional; no additional constraints
	return nil
}

func validateKill(in KillInput) error {
	if !in.SkipConfirmation {
		return Err(E_CONFIRM_REQUIRED, "skipConfirmation=true required for destructive operations in MCP", map[string]any{"hint": "Set skipConfirmation to true to proceed non-interactively."})
	}
	if in.KillAll && in.IsCluster == true {
		return Err(E_INVALID, "killAll and isCluster are mutually exclusive", nil)
	}
	return nil
}
