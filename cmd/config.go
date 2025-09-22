package cmd

import (
	"github.com/hdresearch/vers-cli/internal/runconfig"
	"github.com/spf13/cobra"
)

// applyFlagOverrides applies command-line flag overrides to the config
func applyFlagOverrides(cmd *cobra.Command, config *runconfig.Config) {
	// Override memory size if flag is set
	if memSize, _ := cmd.Flags().GetInt64("mem-size"); memSize > 0 {
		config.Machine.MemSizeMib = memSize
	}

	// Override vcpu count if flag is set
	if vcpuCount, _ := cmd.Flags().GetInt64("vcpu-count"); vcpuCount > 0 {
		config.Machine.VcpuCount = vcpuCount
	}

	// Override rootfs name if flag is set
	if rootfs, _ := cmd.Flags().GetString("rootfs"); rootfs != "" {
		config.Rootfs.Name = rootfs
	}

	// Override kernel name if flag is set
	if kernel, _ := cmd.Flags().GetString("kernel"); kernel != "" {
		config.Kernel.Name = kernel
	}

	if dockerfile, _ := cmd.Flags().GetString("dockerfile"); dockerfile != "" {
		config.Builder.Dockerfile = dockerfile
	}

	// Override filesystem sizes if flags are set
	if fsCluster, _ := cmd.Flags().GetInt64("fs-size-cluster"); fsCluster > 0 {
		config.Machine.FsSizeClusterMib = fsCluster
	}
	if fsVm, _ := cmd.Flags().GetInt64("fs-size-vm"); fsVm > 0 {
		config.Machine.FsSizeVmMib = fsVm
	}
}
