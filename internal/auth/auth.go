package auth

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

// Config represents the structure of the .versrc file
type Config struct {
	APIKey string `json:"apiKey"`
}

// GetConfigPath returns the path to the .versrc file in the user's home directory
func GetConfigPath() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("failed to get user home directory: %w", err)
	}
	return filepath.Join(home, ".versrc"), nil
}

// LoadConfig loads the configuration from the .versrc file
func LoadConfig() (*Config, error) {
	configPath, err := GetConfigPath()
	if err != nil {
		return nil, err
	}

	// If the file doesn't exist, return empty config
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		return &Config{}, nil
	}

	data, err := os.ReadFile(configPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	var config Config
	if len(data) == 0 {
		return &config, nil
	}

	err = json.Unmarshal(data, &config)
	if err != nil {
		return nil, fmt.Errorf("failed to parse config file: %w", err)
	}

	return &config, nil
}

// SaveConfig saves the configuration to the .versrc file
func SaveConfig(config *Config) error {
	configPath, err := GetConfigPath()
	if err != nil {
		return err
	}

	data, err := json.MarshalIndent(config, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}

	err = os.WriteFile(configPath, data, 0600) // Use 0600 for security (user read/write only)
	if err != nil {
		return fmt.Errorf("failed to write config file: %w", err)
	}

	return nil
}

// GetAPIKey retrieves the API key from the config file
func GetAPIKey() (string, error) {
	config, err := LoadConfig()
	if err != nil {
		return "", err
	}
	return config.APIKey, nil
}

// SaveAPIKey saves the API key to the config file
func SaveAPIKey(apiKey string) error {
	config, err := LoadConfig()
	if err != nil {
		return err
	}
	
	config.APIKey = apiKey
	return SaveConfig(config)
}

// HasAPIKey checks if an API key is present
func HasAPIKey() (bool, error) {
	config, err := LoadConfig()
	if err != nil {
		return false, err
	}
	return config.APIKey != "", nil
}

// PromptForLogin creates a helper function that checks for API key and prompts for login if not found
func PromptForLogin() error {
	fmt.Println("No API key found. Please run 'vers login' to authenticate.")
	return nil
} 