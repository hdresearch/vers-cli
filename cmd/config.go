package cmd

import (
	"fmt"
	"os"

	"github.com/BurntSushi/toml"
	"github.com/spf13/cobra"
)

// Config represents the structure of vers.toml
type Config struct {
	Machine MachineConfig `toml:"machine"`
	Rootfs  RootfsConfig  `toml:"rootfs"`
	Builder BuilderConfig `toml:"builder"`
	Kernel  KernelConfig  `toml:"kernel"`
}

type MachineConfig struct {
	MemSizeMib       int64 `toml:"mem_size_mib"`
	VcpuCount        int64 `toml:"vcpu_count"`
	FsSizeClusterMib int64 `toml:"fs_size_cluster_mib"`
	FsSizeVmMib      int64 `toml:"fs_size_vm_mib"`
}

type RootfsConfig struct {
	Name string `toml:"name"`
}

type BuilderConfig struct {
	Name       string `toml:"name"`
	Dockerfile string `toml:"dockerfile"`
}

type KernelConfig struct {
	Name string `toml:"name"`
}

// DefaultConfig returns a config with default values
func DefaultConfig() *Config {
	return &Config{
		Machine: MachineConfig{
			MemSizeMib: 512,
			VcpuCount:  1,
			// Provide safe filesystem defaults to avoid backend errors when vers.toml is missing
			FsSizeClusterMib: 1024,
			FsSizeVmMib:      512,
		},
		Rootfs: RootfsConfig{
			Name: "default",
		},
		Builder: BuilderConfig{
			Name:       "docker",
			Dockerfile: "Dockerfile",
		},
		Kernel: KernelConfig{
			Name: "default.bin",
		},
	}
}

// loadConfig loads the configuration from vers.toml or returns defaults
func loadConfig() (*Config, error) {
	config := DefaultConfig()

	// Check if vers.toml exists
	if _, err := os.Stat("vers.toml"); os.IsNotExist(err) {
		fmt.Println("Warning: vers.toml not found, using default configuration")
		return config, nil
	}

	// Read and parse the toml file
	if _, err := toml.DecodeFile("vers.toml", config); err != nil {
		return nil, fmt.Errorf("error parsing vers.toml: %w", err)
	}

	return config, nil
}

// applyFlagOverrides applies command-line flag overrides to the config
func applyFlagOverrides(cmd *cobra.Command, config *Config) {
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
