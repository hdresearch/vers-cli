package mcp

// StatusInput is the input schema for vers.status tool.
type StatusInput struct {
	Target string `json:"target,omitempty" jsonschema:"VM ID or alias for VM-specific status"`
}

// RunInput defines inputs for vers.run
type RunInput struct {
	MemSizeMib       int64  `json:"memSizeMib,omitempty" jsonschema:"VM memory size in MiB"`
	VcpuCount        int64  `json:"vcpuCount,omitempty" jsonschema:"Number of vCPUs"`
	RootfsName       string `json:"rootfsName,omitempty" jsonschema:"Rootfs image name"`
	KernelName       string `json:"kernelName,omitempty" jsonschema:"Kernel image name"`
	FsSizeClusterMib int64  `json:"fsSizeClusterMib,omitempty" jsonschema:"Cluster filesystem size in MiB"`
	FsSizeVmMib      int64  `json:"fsSizeVmMib,omitempty" jsonschema:"VM filesystem size in MiB"`
	ClusterAlias     string `json:"clusterAlias,omitempty" jsonschema:"Alias for the new cluster"`
	VMAlias          string `json:"vmAlias,omitempty" jsonschema:"Alias for the root VM"`
}

// ExecuteInput defines inputs for vers.execute
type ExecuteInput struct {
	Target         string   `json:"target,omitempty" jsonschema:"VM ID or alias; defaults to HEAD"`
	Command        []string `json:"command" jsonschema:"Command and args to run"`
	TimeoutSeconds int      `json:"timeoutSeconds,omitempty" jsonschema:"Execution timeout in seconds"`
}

// BranchInput defines inputs for vers.branch
type BranchInput struct {
	Target   string `json:"target,omitempty" jsonschema:"Source VM ID or alias; defaults to HEAD"`
	Alias    string `json:"alias,omitempty" jsonschema:"Alias for the new VM"`
	Checkout bool   `json:"checkout,omitempty" jsonschema:"Switch HEAD to the new VM after creation"`
}

// KillInput defines inputs for vers.kill
type KillInput struct {
	Targets          []string `json:"targets,omitempty" jsonschema:"VM identifiers; empty means HEAD VM"`
	SkipConfirmation bool     `json:"skipConfirmation,omitempty" jsonschema:"Required for destructive operations in MCP"`
	Recursive        bool     `json:"recursive,omitempty" jsonschema:"Delete all descendants"`
}
