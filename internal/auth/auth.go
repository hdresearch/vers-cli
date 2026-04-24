package auth

import (
	"crypto/rand"
	"encoding/json"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"strings"

	"github.com/hdresearch/vers-sdk-go/option"
)

const DEFAULT_VERS_URL_STR = "https://api.vers.sh"

// Config represents the structure of the .versrc file
type Config struct {
	APIKey      string           `json:"apiKey"`
	Email       string           `json:"email,omitempty"`
	SSHKeyPath  string           `json:"sshKeyPath,omitempty"`
	UserID      string           `json:"userId,omitempty"`
	OrgID       string           `json:"orgId,omitempty"`
	AnonymousID string           `json:"anonymousId,omitempty"`
	DeviceID    string           `json:"deviceId,omitempty"`
	Telemetry   *TelemetryConfig `json:"telemetry,omitempty"`
}

type TelemetryConfig struct {
	Enabled            *bool `json:"enabled,omitempty"`
	NoticeAcknowledged bool  `json:"noticeAcknowledged,omitempty"`
}

func NewTelemetryID() string {
	var b [16]byte
	_, _ = rand.Read(b[:])
	b[6] = (b[6] & 0x0f) | 0x40
	b[8] = (b[8] & 0x3f) | 0x80
	return fmt.Sprintf("%08x-%04x-%04x-%04x-%012x",
		b[0:4],
		b[4:6],
		b[6:8],
		b[8:10],
		b[10:16],
	)
}

func ClearUserIdentity(config *Config) {
	if config == nil {
		return
	}
	config.UserID = ""
	config.OrgID = ""
	config.Email = ""
}

func EnsureTelemetryConfig(config *Config) *TelemetryConfig {
	if config.Telemetry == nil {
		config.Telemetry = &TelemetryConfig{}
	}
	return config.Telemetry
}

func EffectiveTelemetryStatus(config *Config) (enabled bool, reason string) {
	if value := strings.ToLower(strings.TrimSpace(os.Getenv("DO_NOT_TRACK"))); value == "1" || value == "true" {
		return false, "disabled by DO_NOT_TRACK"
	}
	if config != nil && config.Telemetry != nil && config.Telemetry.Enabled != nil {
		if *config.Telemetry.Enabled {
			return true, "enabled in .versrc"
		}
		return false, "disabled in .versrc"
	}
	return true, "enabled by default"
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
	errorMsg := "No API key found. Please run 'vers signup' or 'vers login' to authenticate."
	fmt.Println(errorMsg)
	return nil
}

// GetVersUrl returns the full URL with protocol validation
func GetVersUrl() (*url.URL, error) {
	versUrlStr := os.Getenv("VERS_URL")
	if strings.TrimSpace(versUrlStr) == "" {
		versUrlStr = DEFAULT_VERS_URL_STR
	}

	versUrl, err := url.Parse(versUrlStr)
	if err != nil {
		return nil, err
	} else if versUrl.Scheme != "http" && versUrl.Scheme != "https" {
		return nil, fmt.Errorf("invalid VERS_URL %s; URL must include scheme http:// or https://", versUrl)
	}

	return versUrl, nil
}

// GetSiteURL returns the main site URL corresponding to the configured API URL.
// For example:
//   - https://api.vers.sh -> https://vers.sh
//   - https://api.staging.vers.sh -> https://staging.vers.sh
func GetSiteURL() (*url.URL, error) {
	versURL, err := GetVersUrl()
	if err != nil {
		return nil, err
	}

	siteURL := *versURL
	host := versURL.Hostname()
	if strings.HasPrefix(host, "api.") {
		host = strings.TrimPrefix(host, "api.")
	}
	if port := versURL.Port(); port != "" {
		siteURL.Host = host + ":" + port
	} else {
		siteURL.Host = host
	}
	return &siteURL, nil
}

// GetClientOptions returns the options for the SDK client
func GetClientOptions() ([]option.RequestOption, error) {
	clientOptions := []option.RequestOption{}

	// Get the full URL with appropriate protocol
	versUrl, err := GetVersUrl()
	if err != nil {
		return nil, err
	}

	// Set the base URL with protocol and API version path
	baseURL := strings.TrimSuffix(versUrl.String(), "/")
	clientOptions = append(clientOptions, option.WithBaseURL(baseURL))

	// Allow overriding the Host header for testing against dev instances
	if hostHeader := os.Getenv("VERS_HOST_HEADER"); hostHeader != "" {
		clientOptions = append(clientOptions, option.WithHeader("Host", hostHeader))
		if os.Getenv("VERS_VERBOSE") == "true" {
			fmt.Printf("[DEBUG] Using Host header: %s\n", hostHeader)
		}
	}

	if os.Getenv("VERS_VERBOSE") == "true" {
		fmt.Printf("[DEBUG] Using API endpoint: %s\n", versUrl)
		// Enable SDK debug logging (logs HTTP requests/responses)
		clientOptions = append(clientOptions, option.WithDebugLog(nil))
	}

	return clientOptions, nil
}

// GetVMDomain derives the VM SSH domain from the configured VERS_URL.
// For example:
//   - "https://api.vers.sh"         -> "vm.vers.sh"
//   - "https://api.staging.vers.sh" -> "vm.staging.vers.sh"
func GetVMDomain() string {
	versUrl, err := GetVersUrl()
	if err != nil {
		return "vm.vers.sh" // safe fallback to production
	}
	host := versUrl.Hostname()
	if strings.HasPrefix(host, "api.") {
		return "vm." + strings.TrimPrefix(host, "api.")
	}
	// Fallback: assume production domain
	return "vm.vers.sh"
}

// Backward-compatibility checks removed; the CLI only targets DEFAULT_VERS_URL (or VERS_URL override).
