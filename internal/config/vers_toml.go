package config

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/BurntSushi/toml"
)

// Config represents the structure of vers.toml
type Config struct {
	Meta    MetaConfig               `toml:"meta"`
	Build   BuildConfig              `toml:"build"`
	Deploy  DeployConfig             `toml:"deploy"`
	Run     RunConfig                `toml:"run"`
	Env     map[string]string        `toml:"env"`
	Machine map[string]MachineConfig `toml:"machine"`
}

// MetaConfig holds project metadata
type MetaConfig struct {
	Project string `toml:"project"`
	Type    string `toml:"type"`
}

// BuildConfig holds build configuration
type BuildConfig struct {
	Builder      string `toml:"builder"`
	BuildCommand string `toml:"build_command"`
}

// DeployConfig holds deployment configuration
type DeployConfig struct {
	Platform string `toml:"platform"`
}

// RunConfig holds runtime configuration
type RunConfig struct {
	Command     string   `toml:"command"`
	EntryPoints []string `toml:"entry_points"`
}

// MachineConfig holds configuration for a specific machine
type MachineConfig struct {
	Name  string `toml:"name"`
	Image string `toml:"image"`
	IP    string `toml:"ip"`
	Port  string `toml:"port"`
}

// LoadConfig loads the vers.toml configuration file
func LoadConfig(path string) (*Config, error) {
	var config Config
	if _, err := toml.DecodeFile(path, &config); err != nil {
		return nil, fmt.Errorf("error parsing %s: %w", path, err)
	}
	return &config, nil
}

// FindConfig looks for vers.toml in the current directory or parent directories
func FindConfig() (string, *Config, error) {
	dir, err := os.Getwd()
	if err != nil {
		return "", nil, err
	}

	for {
		configPath := filepath.Join(dir, "vers.toml")
		if _, err := os.Stat(configPath); err == nil {
			config, err := LoadConfig(configPath)
			if err != nil {
				return configPath, nil, err
			}
			return configPath, config, nil
		}

		parent := filepath.Dir(dir)
		if parent == dir {
			break
		}
		dir = parent
	}

	return "", nil, fmt.Errorf("vers.toml not found in current directory or any parent directory")
}
