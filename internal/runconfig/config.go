package runconfig

import (
	"fmt"
	"os"

	"github.com/BurntSushi/toml"
)

// Config represents the structure of vers.toml for runtime/build needs.
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

// Default returns a config with safe defaults.
func Default() *Config {
	return &Config{
		Machine: MachineConfig{
			MemSizeMib:       512,
			VcpuCount:        1,
			FsSizeClusterMib: 1024,
			FsSizeVmMib:      512,
		},
		Rootfs: RootfsConfig{Name: "default"},
		Builder: BuilderConfig{
			Name:       "docker",
			Dockerfile: "Dockerfile",
		},
		Kernel: KernelConfig{Name: "default.bin"},
	}
}

// Load loads vers.toml if present, otherwise returns Default() and prints a warning.
func Load() (*Config, error) {
	cfg := Default()
	if _, err := os.Stat("vers.toml"); os.IsNotExist(err) {
		fmt.Println("Warning: vers.toml not found, using default configuration")
		return cfg, nil
	}
	if _, err := toml.DecodeFile("vers.toml", cfg); err != nil {
		return nil, fmt.Errorf("error parsing vers.toml: %w", err)
	}
	return cfg, nil
}
