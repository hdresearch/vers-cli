package auth

import (
	"os"
	"path/filepath"
	"testing"
)

func TestReadAndValidateSSHPublicKey_Valid(t *testing.T) {
	tests := []struct {
		name    string
		content string
	}{
		{"ed25519", "ssh-ed25519 AAAAC3NzaC1lZDI1NTE5AAAAIExampleKeyDataHere user@host"},
		{"rsa", "ssh-rsa AAAAB3NzaC1yc2EAAAADAQABAAABgQExample user@host"},
		{"ecdsa", "ecdsa-sha2-nistp256 AAAAE2VjZHNhLXNoYTItbmlzdHAyNTY= user@host"},
		{"no comment", "ssh-ed25519 AAAAC3NzaC1lZDI1NTE5AAAAIExampleKeyDataHere"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			path := filepath.Join(t.TempDir(), "id_test.pub")
			os.WriteFile(path, []byte(tt.content), 0644)

			key, err := ReadAndValidateSSHPublicKey(path)
			if err != nil {
				t.Fatalf("expected no error, got: %v", err)
			}
			if key != tt.content {
				t.Errorf("expected key %q, got %q", tt.content, key)
			}
		})
	}
}

func TestReadAndValidateSSHPublicKey_Invalid(t *testing.T) {
	tests := []struct {
		name    string
		setup   func(dir string) string // returns path
		wantErr string
	}{
		{
			name: "file not found",
			setup: func(dir string) string {
				return filepath.Join(dir, "nonexistent")
			},
			wantErr: "not found",
		},
		{
			name: "is a directory",
			setup: func(dir string) string {
				p := filepath.Join(dir, "subdir")
				os.Mkdir(p, 0755)
				return p
			},
			wantErr: "directory",
		},
		{
			name: "empty file",
			setup: func(dir string) string {
				p := filepath.Join(dir, "empty.pub")
				os.WriteFile(p, []byte(""), 0644)
				return p
			},
			wantErr: "empty",
		},
		{
			name: "whitespace only",
			setup: func(dir string) string {
				p := filepath.Join(dir, "blank.pub")
				os.WriteFile(p, []byte("   \n  \n"), 0644)
				return p
			},
			wantErr: "empty",
		},
		{
			name: "single field no base64",
			setup: func(dir string) string {
				p := filepath.Join(dir, "bad.pub")
				os.WriteFile(p, []byte("ssh-ed25519"), 0644)
				return p
			},
			wantErr: "invalid SSH public key format",
		},
		{
			name: "unknown key type",
			setup: func(dir string) string {
				p := filepath.Join(dir, "bad.pub")
				os.WriteFile(p, []byte("ssh-dsa AAAAB3NzaC1kc3MAAACB user@host"), 0644)
				return p
			},
			wantErr: "unrecognized SSH key type",
		},
		{
			name: "private key (too large isn't the check here, but wrong format)",
			setup: func(dir string) string {
				p := filepath.Join(dir, "id_rsa")
				os.WriteFile(p, []byte("-----BEGIN OPENSSH PRIVATE KEY-----\ndata\n-----END OPENSSH PRIVATE KEY-----"), 0644)
				return p
			},
			wantErr: "unrecognized SSH key type",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dir := t.TempDir()
			path := tt.setup(dir)

			_, err := ReadAndValidateSSHPublicKey(path)
			if err == nil {
				t.Fatal("expected error, got nil")
			}
			if !contains(err.Error(), tt.wantErr) {
				t.Errorf("expected error containing %q, got: %v", tt.wantErr, err)
			}
		})
	}
}

func TestReadAndValidateSSHPublicKey_TooLarge(t *testing.T) {
	path := filepath.Join(t.TempDir(), "big.pub")
	// 17KB file
	os.WriteFile(path, make([]byte, 17*1024), 0644)

	_, err := ReadAndValidateSSHPublicKey(path)
	if err == nil {
		t.Fatal("expected error for oversized file")
	}
	if !contains(err.Error(), "too large") {
		t.Errorf("expected 'too large' error, got: %v", err)
	}
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && containsStr(s, substr)
}

func containsStr(s, sub string) bool {
	for i := 0; i <= len(s)-len(sub); i++ {
		if s[i:i+len(sub)] == sub {
			return true
		}
	}
	return false
}
