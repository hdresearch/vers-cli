package auth

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/hdresearch/vers-cli/styles"
	"github.com/hdresearch/vers-sdk-go/option"
)

// TODO: Remove backward compatibility after migration period (target: later this week will probably be fine tbh)
// During migration: support both old IP and new domain
const LEGACY_VERS_URL = "13.219.19.157" // Keep for reference during migration
const DEFAULT_VERS_URL = "https://api.vers.sh"

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

// GetAPIKey retrieves the API key from environment variable or config file
func GetAPIKey() (string, error) {
	// First check environment variable
	if apiKey := os.Getenv("VERS_API_KEY"); apiKey != "" {
		return apiKey, nil
	}

	// Fallback to config file
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

// HasAPIKey checks if an API key is present in environment variable or config file
func HasAPIKey() (bool, error) {
	// First check environment variable
	if apiKey := os.Getenv("VERS_API_KEY"); apiKey != "" {
		return true, nil
	}

	// Fallback to config file
	config, err := LoadConfig()
	if err != nil {
		return false, err
	}
	return config.APIKey != "", nil
}

// PromptForLogin creates a helper function that checks for API key and prompts for login if not found
func PromptForLogin() error {
	errorMsg := styles.ErrorTextStyle.Render("No API key found. Please run 'vers login' to authenticate.")
	fmt.Println(errorMsg)
	return nil
}

// GetVersUrl returns the full URL with protocol validation
func GetVersUrl() (string, error) {
	versUrl := os.Getenv("VERS_URL")

	// If VERS_URL is set, it must include protocol
	if versUrl != "" {
		if strings.HasPrefix(versUrl, "http://") || strings.HasPrefix(versUrl, "https://") {
			return versUrl, nil
		}
		return "", fmt.Errorf("VERS_URL must include protocol (http:// or https://), got: %s", versUrl)
	}

	// Default to https with default URL
	return DEFAULT_VERS_URL, nil
}

// GetVersUrlHost returns just the hostname/IP from the URL (for SSH connections)
func GetVersUrlHost() (string, error) {
	fullUrl, err := GetVersUrl()
	if err != nil {
		return "", err
	}

	// Strip protocol to get just hostname/IP
	if strings.HasPrefix(fullUrl, "http://") {
		return strings.TrimPrefix(fullUrl, "http://"), nil
	} else if strings.HasPrefix(fullUrl, "https://") {
		return strings.TrimPrefix(fullUrl, "https://"), nil
	}

	return fullUrl, nil
}

// GetClientOptions returns the options for the SDK client
// TODO: Simplify after migration period - remove protocol detection logic
func GetClientOptions() []option.RequestOption {
	clientOptions := []option.RequestOption{}

	// Get the full URL with appropriate protocol
	fullUrl, err := GetVersUrl()
	if err != nil {
		fmt.Printf("[ERROR] %v\n", err)
		return nil
	}

	// BACKWARD COMPATIBILITY: Show deprecation notice for legacy endpoint
	// TODO: Remove this logic after migration period
	if versUrlHost, _ := GetVersUrlHost(); versUrlHost == LEGACY_VERS_URL {
		if os.Getenv("VERS_VERBOSE") == "true" {
			fmt.Printf("[DEPRECATED] Using legacy endpoint: %s. Please update to use new API keys from https://vers.sh\n", fullUrl)
		}
	}

	// Set the base URL with protocol
	clientOptions = append(clientOptions, option.WithBaseURL(fullUrl))

	if os.Getenv("VERS_VERBOSE") == "true" {
		fmt.Printf("[DEBUG] Using API endpoint: %s\n", fullUrl)
	}

	return clientOptions
}

// CheckForLegacyKey shows a deprecation notice for legacy keys
// TODO: Remove after migration period
func CheckForLegacyKey() {
	config, err := LoadConfig()
	if err != nil {
		return
	}

	// Show deprecation notice if using legacy endpoint
	if config.APIKey != "" {
		if versUrlHost, _ := GetVersUrlHost(); versUrlHost == LEGACY_VERS_URL {
			fmt.Println("Notice: You're using a legacy API endpoint. Consider generating a new key at https://vers.sh for improved security and features.")
		}
	}
}
