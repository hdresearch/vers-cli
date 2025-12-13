package utils

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

// GetAliasesPath returns the path to the aliases file (~/.vers/aliases.json)
func GetAliasesPath() (string, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("failed to get user home directory: %w", err)
	}

	configDir := filepath.Join(homeDir, ".vers")
	if err := os.MkdirAll(configDir, 0755); err != nil {
		return "", fmt.Errorf("failed to create config directory: %w", err)
	}

	return filepath.Join(configDir, "aliases.json"), nil
}

// LoadAliases loads the alias map from ~/.vers/aliases.json
// Returns an empty map if the file doesn't exist
func LoadAliases() (map[string]string, error) {
	aliasPath, err := GetAliasesPath()
	if err != nil {
		return nil, err
	}

	if _, err := os.Stat(aliasPath); os.IsNotExist(err) {
		return make(map[string]string), nil
	}

	data, err := os.ReadFile(aliasPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read aliases file: %w", err)
	}

	var aliases map[string]string
	if err := json.Unmarshal(data, &aliases); err != nil {
		return nil, fmt.Errorf("failed to parse aliases file: %w", err)
	}

	if aliases == nil {
		aliases = make(map[string]string)
	}

	return aliases, nil
}

// SaveAliases saves the alias map to ~/.vers/aliases.json
func SaveAliases(aliases map[string]string) error {
	aliasPath, err := GetAliasesPath()
	if err != nil {
		return err
	}

	data, err := json.MarshalIndent(aliases, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal aliases: %w", err)
	}

	if err := os.WriteFile(aliasPath, data, 0644); err != nil {
		return fmt.Errorf("failed to write aliases file: %w", err)
	}

	return nil
}

// SetAlias adds or updates an alias mapping
func SetAlias(alias, vmID string) error {
	aliases, err := LoadAliases()
	if err != nil {
		return err
	}

	aliases[alias] = vmID
	return SaveAliases(aliases)
}

// RemoveAlias removes an alias mapping
func RemoveAlias(alias string) error {
	aliases, err := LoadAliases()
	if err != nil {
		return err
	}

	delete(aliases, alias)
	return SaveAliases(aliases)
}

// ResolveAlias checks if the identifier is an alias and returns the VM ID
// If not an alias, returns the identifier unchanged
func ResolveAlias(identifier string) string {
	aliases, err := LoadAliases()
	if err != nil {
		return identifier
	}

	if vmID, ok := aliases[identifier]; ok {
		return vmID
	}

	return identifier
}

// GetAliasByVMID performs a reverse lookup to find an alias for a VM ID
// Returns empty string if no alias exists
func GetAliasByVMID(vmID string) string {
	aliases, err := LoadAliases()
	if err != nil {
		return ""
	}

	for alias, id := range aliases {
		if id == vmID {
			return alias
		}
	}

	return ""
}
