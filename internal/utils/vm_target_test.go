package utils

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLooksLikeVMTarget_UUID(t *testing.T) {
	tests := []struct {
		name   string
		input  string
		expect bool
	}{
		{"valid UUID", "3bfea344-6bf2-4655-be27-64be7b5eb332", true},
		{"valid UUID uppercase", "3BFEA344-6BF2-4655-BE27-64BE7B5EB332", true},
		{"valid UUID mixed case", "3bFeA344-6Bf2-4655-bE27-64bE7b5eB332", true},
		{"zeroed UUID", "00000000-0000-0000-0000-000000000000", true},
		{"not a UUID - command", "echo", false},
		{"not a UUID - command with path", "/usr/bin/cat", false},
		{"not a UUID - partial UUID", "3bfea344-6bf2", false},
		{"not a UUID - jq", "jq", false},
		{"not a UUID - python3", "python3", false},
		{"not a UUID - ls", "ls", false},
		{"not a UUID - bash", "bash", false},
		{"not a UUID - empty", "", false},
		{"not a UUID - UUID without dashes", "3bfea3446bf24655be2764be7b5eb332", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := LooksLikeVMTarget(tt.input)
			if got != tt.expect {
				t.Errorf("LooksLikeVMTarget(%q) = %v, want %v", tt.input, got, tt.expect)
			}
		})
	}
}

func TestLooksLikeVMTarget_Alias(t *testing.T) {
	// Set up a temp home dir with aliases.
	// Must set both HOME (Unix) and USERPROFILE (Windows) since
	// os.UserHomeDir() checks USERPROFILE on Windows.
	tmpHome := t.TempDir()

	origHome := os.Getenv("HOME")
	origUserProfile := os.Getenv("USERPROFILE")
	os.Setenv("HOME", tmpHome)
	os.Setenv("USERPROFILE", tmpHome)
	defer os.Setenv("HOME", origHome)
	defer os.Setenv("USERPROFILE", origUserProfile)

	aliasDir := filepath.Join(tmpHome, ".vers")
	os.MkdirAll(aliasDir, 0755)
	os.WriteFile(filepath.Join(aliasDir, "aliases.json"), []byte(`{"my-dev-vm":"3bfea344-6bf2-4655-be27-64be7b5eb332"}`), 0644)

	tests := []struct {
		name   string
		input  string
		expect bool
	}{
		{"known alias", "my-dev-vm", true},
		{"unknown alias", "no-such-alias", false},
		{"command not alias", "cat", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := LooksLikeVMTarget(tt.input)
			if got != tt.expect {
				t.Errorf("LooksLikeVMTarget(%q) = %v, want %v", tt.input, got, tt.expect)
			}
		})
	}
}
