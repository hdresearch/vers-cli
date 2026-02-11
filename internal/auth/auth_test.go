package auth

import (
	"os"
	"testing"
)

func TestGetVMDomain(t *testing.T) {
	tests := []struct {
		name     string
		versURL  string
		expected string
	}{
		{"production default", "", "vm.vers.sh"},
		{"production explicit", "https://api.vers.sh", "vm.vers.sh"},
		{"staging", "https://api.staging.vers.sh", "vm.staging.vers.sh"},
		{"custom", "https://api.dev.example.com", "vm.dev.example.com"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if tc.versURL != "" {
				os.Setenv("VERS_URL", tc.versURL)
				defer os.Unsetenv("VERS_URL")
			} else {
				os.Unsetenv("VERS_URL")
			}

			got := GetVMDomain()
			if got != tc.expected {
				t.Errorf("GetVMDomain() = %q, want %q", got, tc.expected)
			}
		})
	}
}
