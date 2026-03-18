package cmd

import (
	"testing"
)

// TestSubcommandRouting verifies that Cobra routes subcommands correctly
// instead of treating them as positional arguments to the parent command.
//
// This catches the bug where `vers commit list` was parsed as
// `vers commit [vm-id=list]` instead of routing to the `list` subcommand.
func TestSubcommandRouting(t *testing.T) {
	// All parent commands that have both subcommands and a RunE.
	// Each entry: parent command name, subcommand names that must route correctly.
	tests := []struct {
		parent      string
		subcommands []string
	}{
		{"commit", []string{"create", "list", "delete", "history", "publish", "unpublish"}},
		{"tag", []string{"create", "list", "get", "update", "delete"}},
	}

	for _, tt := range tests {
		parent, _, err := rootCmd.Find([]string{tt.parent})
		if err != nil {
			t.Fatalf("could not find parent command %q: %v", tt.parent, err)
		}
		if parent == nil {
			t.Fatalf("parent command %q is nil", tt.parent)
		}

		for _, sub := range tt.subcommands {
			t.Run(tt.parent+" "+sub, func(t *testing.T) {
				cmd, args, err := rootCmd.Find([]string{tt.parent, sub})
				if err != nil {
					t.Fatalf("Find(%q %q) returned error: %v", tt.parent, sub, err)
				}
				if cmd == nil {
					t.Fatalf("Find(%q %q) returned nil command", tt.parent, sub)
				}
				if cmd.Name() != sub {
					t.Errorf("expected command %q but got %q (args=%v) — subcommand was swallowed as a positional arg", sub, cmd.Name(), args)
				}
				if len(args) != 0 {
					t.Errorf("expected no leftover args, got %v — %q was not recognized as a subcommand", args, sub)
				}
			})
		}
	}
}
