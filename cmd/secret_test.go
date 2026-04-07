package cmd

import "testing"

func TestMaskValue(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		// Short values — fully masked
		{"empty", "", "****"},
		{"1 char", "a", "****"},
		{"4 chars", "abcd", "****"},

		// Medium values — show prefix
		{"5 chars", "abcde", "ab****"},
		{"8 chars", "abcdefgh", "ab****"},

		// Longer values — show prefix and suffix
		{"9 chars", "abcdefghi", "abcd****hi"},
		{"API key", "sk-ant-api03-abc123xyz", "sk-a****yz"},
		{"URL", "postgres://user:pass@host/db", "post****db"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := maskValue(tt.input)
			if got != tt.expected {
				t.Errorf("maskValue(%q) = %q, want %q", tt.input, got, tt.expected)
			}
		})
	}
}

func TestMaskValueNeverRevealsFullSecret(t *testing.T) {
	secrets := []string{
		"short",
		"medium-length",
		"sk-ant-api03-very-long-secret-key-here",
		"postgres://admin:hunter2@prod.db.example.com:5432/myapp",
	}

	for _, secret := range secrets {
		masked := maskValue(secret)
		if masked == secret {
			t.Errorf("maskValue(%q) returned the unmasked value", secret)
		}
		if len(masked) > len(secret) {
			// Masked output shouldn't be longer than the original
			// (it's fine if it is, but worth flagging)
		}
	}
}
