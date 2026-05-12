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

func TestGetOrgName(t *testing.T) {
	// Save and restore HOME / VERS_ORG so this test is hermetic.
	origHome := os.Getenv("HOME")
	origOrg := os.Getenv("VERS_ORG")
	t.Cleanup(func() {
		os.Setenv("HOME", origHome)
		if origOrg == "" {
			os.Unsetenv("VERS_ORG")
		} else {
			os.Setenv("VERS_ORG", origOrg)
		}
	})

	tmp := t.TempDir()
	os.Setenv("HOME", tmp)
	os.Unsetenv("VERS_ORG")

	// 1. No config file, no env var → empty.
	got, err := GetOrgName()
	if err != nil {
		t.Fatalf("GetOrgName with no config: %v", err)
	}
	if got != "" {
		t.Errorf("expected empty, got %q", got)
	}

	// 2. Persist via SaveAuth, read back.
	if err := SaveAuth("test-key", "acme", "org-uuid-1"); err != nil {
		t.Fatalf("SaveAuth: %v", err)
	}
	got, err = GetOrgName()
	if err != nil {
		t.Fatalf("GetOrgName after SaveAuth: %v", err)
	}
	if got != "acme" {
		t.Errorf("expected acme, got %q", got)
	}

	// 3. Env var wins over persisted value.
	os.Setenv("VERS_ORG", "override-org")
	got, err = GetOrgName()
	if err != nil {
		t.Fatalf("GetOrgName with env override: %v", err)
	}
	if got != "override-org" {
		t.Errorf("expected override-org, got %q", got)
	}

	// 4. Empty SaveAuth values don't clobber persisted ones.
	os.Unsetenv("VERS_ORG")
	if err := SaveAuth("new-key", "", ""); err != nil {
		t.Fatalf("SaveAuth empty org: %v", err)
	}
	cfg, err := LoadConfig()
	if err != nil {
		t.Fatalf("LoadConfig: %v", err)
	}
	if cfg.APIKey != "new-key" {
		t.Errorf("expected new-key, got %q", cfg.APIKey)
	}
	if cfg.OrgName != "acme" {
		t.Errorf("expected acme preserved, got %q", cfg.OrgName)
	}
	if cfg.OrgID != "org-uuid-1" {
		t.Errorf("expected org-uuid-1 preserved, got %q", cfg.OrgID)
	}
}
