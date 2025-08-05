package config

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

// CLIConfig represents the global CLI configuration stored in ~/.vers/config.json
type CLIConfig struct {
	UpdateCheck UpdateCheckConfig `json:"update_check"`
}

// UpdateCheckConfig holds update checking state
type UpdateCheckConfig struct {
	LastCheck     time.Time `json:"last_check"`
	NextCheck     time.Time `json:"next_check"`
	CheckInterval int64     `json:"check_interval"` // in seconds, default 3600 (1 hour)
}

// GetCLIConfigPath returns the path to the CLI config file
func GetCLIConfigPath() (string, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("failed to get user home directory: %w", err)
	}

	configDir := filepath.Join(homeDir, ".vers")
	if err := os.MkdirAll(configDir, 0755); err != nil {
		return "", fmt.Errorf("failed to create config directory: %w", err)
	}

	return filepath.Join(configDir, "config.json"), nil
}

// LoadCLIConfig loads the CLI configuration from ~/.vers/config.json
func LoadCLIConfig() (*CLIConfig, error) {
	configPath, err := GetCLIConfigPath()
	if err != nil {
		return nil, err
	}

	// If config file doesn't exist, return default config
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		now := time.Now()
		return &CLIConfig{
			UpdateCheck: UpdateCheckConfig{
				LastCheck:     now,
				NextCheck:     now,  // Check immediately on first run
				CheckInterval: 3600, // 1 hour
			},
		}, nil
	}

	data, err := os.ReadFile(configPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read CLI config: %w", err)
	}

	var config CLIConfig
	if err := json.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("failed to parse CLI config: %w", err)
	}

	// Ensure we have default values for missing fields
	if config.UpdateCheck.CheckInterval == 0 {
		config.UpdateCheck.CheckInterval = 3600
	}

	// Fix zero times that might exist from old configs
	if config.UpdateCheck.NextCheck.IsZero() {
		config.UpdateCheck.NextCheck = time.Now()
	}

	return &config, nil
}

// SaveCLIConfig saves the CLI configuration to ~/.vers/config.json
func SaveCLIConfig(config *CLIConfig) error {
	configPath, err := GetCLIConfigPath()
	if err != nil {
		return err
	}

	data, err := json.MarshalIndent(config, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal CLI config: %w", err)
	}

	if err := os.WriteFile(configPath, data, 0644); err != nil {
		return fmt.Errorf("failed to write CLI config: %w", err)
	}

	return nil
}

// ShouldCheckForUpdate returns true if it's time to check for updates
func (c *CLIConfig) ShouldCheckForUpdate() bool {
	return time.Now().After(c.UpdateCheck.NextCheck)
}

// SetNextCheckTime sets the next check time based on the interval
func (c *CLIConfig) SetNextCheckTime() {
	c.UpdateCheck.LastCheck = time.Now()
	c.UpdateCheck.NextCheck = time.Now().Add(time.Duration(c.UpdateCheck.CheckInterval) * time.Second)
}
