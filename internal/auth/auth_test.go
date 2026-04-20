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

func TestGetVersLandingURL(t *testing.T) {
	tests := []struct {
		name      string
		envValue  string // "" means unset
		expectURL string // empty if we expect an error
		expectErr bool
	}{
		{"default when env unset", "", DEFAULT_VERS_LANDING_URL_STR, false},
		{"default when env empty string", "", DEFAULT_VERS_LANDING_URL_STR, false},
		{"default when env whitespace", "   ", DEFAULT_VERS_LANDING_URL_STR, false},
		{"staging override", "https://staging.vers.sh", "https://staging.vers.sh", false},
		{"local dev override", "http://localhost:3001", "http://localhost:3001", false},
		{"missing scheme errors", "vers.sh", "", true},
		{"unsupported scheme errors", "ftp://vers.sh", "", true},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if tc.envValue != "" {
				os.Setenv("VERS_LANDING_URL", tc.envValue)
				defer os.Unsetenv("VERS_LANDING_URL")
			} else {
				os.Unsetenv("VERS_LANDING_URL")
			}

			got, err := GetVersLandingURL()
			if tc.expectErr {
				if err == nil {
					t.Fatalf("expected error for env=%q, got URL %v", tc.envValue, got)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got.String() != tc.expectURL {
				t.Errorf("GetVersLandingURL() = %q, want %q", got.String(), tc.expectURL)
			}
		})
	}
}

// TestGetVersLandingURL_IndependentOfVersURL verifies that the landing URL
// is NOT derived from VERS_URL (that was the old heuristic-host-munging
// behavior that we replaced with an explicit env var).
func TestGetVersLandingURL_IndependentOfVersURL(t *testing.T) {
	os.Setenv("VERS_URL", "https://api.staging.vers.sh")
	defer os.Unsetenv("VERS_URL")
	os.Unsetenv("VERS_LANDING_URL")

	got, err := GetVersLandingURL()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got.String() != DEFAULT_VERS_LANDING_URL_STR {
		t.Errorf("expected default landing URL regardless of VERS_URL, got %q", got.String())
	}
}
