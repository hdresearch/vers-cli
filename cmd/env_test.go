package cmd

import "testing"

func TestIsValidEnvKey(t *testing.T) {
	tests := []struct {
		key   string
		valid bool
	}{
		// Valid keys
		{"FOO", true},
		{"_BAR", true},
		{"DATABASE_URL", true},
		{"API_KEY_V2", true},
		{"a", true},
		{"_", true},
		{"A1_B2", true},
		
		// Invalid keys
		{"", false},                    // empty
		{"1BAD", false},               // starts with digit
		{"BAD KEY", false},            // contains space
		{"BAD-KEY", false},            // contains dash
		{"BAD.KEY", false},            // contains dot
		{"BAD;KEY", false},            // contains semicolon
		{"café", false},               // non-ASCII characters
		{string(make([]byte, 257)), false}, // too long
	}

	for _, tt := range tests {
		t.Run(tt.key, func(t *testing.T) {
			got := isValidEnvKey(tt.key)
			if got != tt.valid {
				t.Errorf("isValidEnvKey(%q) = %v, want %v", tt.key, got, tt.valid)
			}
		})
	}
}

func TestIsValidEnvKeyEdgeCases(t *testing.T) {
	// Test maximum length (256 characters)
	validMax := string(make([]byte, 256))
	for i := range validMax {
		if i == 0 {
			validMax = "A" + validMax[1:]
		} else {
			validMax = validMax[:i] + "a" + validMax[i+1:]
		}
	}
	if !isValidEnvKey(validMax) {
		t.Errorf("isValidEnvKey with 256 chars should be valid")
	}

	// Test over maximum length (257 characters)
	invalidMax := validMax + "X"
	if isValidEnvKey(invalidMax) {
		t.Errorf("isValidEnvKey with 257 chars should be invalid")
	}

	// Test all valid characters
	validChars := "ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789_"
	if !isValidEnvKey(validChars) {
		t.Errorf("isValidEnvKey with all valid chars should be valid")
	}
}