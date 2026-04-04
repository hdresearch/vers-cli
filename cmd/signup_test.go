package cmd

import (
	"testing"
)

// TestSignupCommandExists verifies the signup command is registered and routable.
func TestSignupCommandExists(t *testing.T) {
	cmd, args, err := rootCmd.Find([]string{"signup"})
	if err != nil {
		t.Fatalf("Find(signup) returned error: %v", err)
	}
	if cmd == nil {
		t.Fatal("Find(signup) returned nil command")
	}
	if cmd.Name() != "signup" {
		t.Errorf("expected command name %q, got %q", "signup", cmd.Name())
	}
	if len(args) != 0 {
		t.Errorf("expected no leftover args, got %v", args)
	}
}

// TestSignupGitFlagDefault verifies --git defaults to true.
func TestSignupGitFlagDefault(t *testing.T) {
	cmd, _, err := rootCmd.Find([]string{"signup"})
	if err != nil {
		t.Fatalf("Find(signup) returned error: %v", err)
	}

	flag := cmd.Flags().Lookup("git")
	if flag == nil {
		t.Fatal("signup command has no --git flag")
	}
	if flag.DefValue != "true" {
		t.Errorf("expected --git default value %q, got %q", "true", flag.DefValue)
	}
}

// TestSignupGitFlagCanBeDisabled verifies --git=false is accepted.
func TestSignupGitFlagCanBeDisabled(t *testing.T) {
	cmd, _, err := rootCmd.Find([]string{"signup"})
	if err != nil {
		t.Fatalf("Find(signup) returned error: %v", err)
	}

	err = cmd.Flags().Set("git", "false")
	if err != nil {
		t.Fatalf("failed to set --git=false: %v", err)
	}

	val, err := cmd.Flags().GetBool("git")
	if err != nil {
		t.Fatalf("failed to get --git value: %v", err)
	}
	if val != false {
		t.Error("expected --git to be false after setting")
	}

	// Reset for other tests
	cmd.Flags().Set("git", "true")
}

// TestSignupNoUnexpectedFlags ensures signup doesn't accept random flags.
func TestSignupNoUnexpectedFlags(t *testing.T) {
	cmd, _, err := rootCmd.Find([]string{"signup"})
	if err != nil {
		t.Fatalf("Find(signup) returned error: %v", err)
	}

	flag := cmd.Flags().Lookup("token")
	if flag != nil {
		t.Error("signup should not have a --token flag (that's login's job)")
	}
}
