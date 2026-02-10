package cmd

import (
	"os"
	"testing"

	"github.com/hdresearch/vers-cli/internal/utils"
	"github.com/spf13/cobra"
)

// TestExecuteCommandArgumentParsing tests the argument parsing logic for execute
func TestExecuteCommandArgumentParsing(t *testing.T) {
	tests := []struct {
		name            string
		args            []string
		expectError     bool
		expectedTarget  string
		expectedCommand []string
		errorMessage    string
	}{
		{
			name:            "2+ args: first is target, rest is command",
			args:            []string{"my-vm", "echo", "hello"},
			expectError:     false,
			expectedTarget:  "my-vm",
			expectedCommand: []string{"echo", "hello"},
		},
		{
			name:            "2 args: target and single command",
			args:            []string{"my-vm", "ls"},
			expectError:     false,
			expectedTarget:  "my-vm",
			expectedCommand: []string{"ls"},
		},
		{
			name:            "1 arg: command only, uses HEAD",
			args:            []string{"echo hello"},
			expectError:     false,
			expectedTarget:  "",
			expectedCommand: []string{"echo hello"},
		},
		{
			name:         "0 args: error",
			args:         []string{},
			expectError:  true,
			errorMessage: "requires at least 1 arg(s)",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var capturedTarget string
			var capturedCommand []string

			cmd := &cobra.Command{
				Use:  "execute [vm-id|alias] <command> [args...]",
				Args: cobra.MinimumNArgs(1),
				RunE: func(cmd *cobra.Command, args []string) error {
					if len(args) == 1 {
						capturedTarget = ""
						capturedCommand = args
					} else {
						capturedTarget = args[0]
						capturedCommand = args[1:]
					}
					return nil
				},
			}

			cmd.SetArgs(tt.args)
			err := cmd.Execute()

			if tt.expectError {
				if err == nil {
					t.Errorf("Expected error but got none")
				}
				return
			}

			if err != nil {
				t.Fatalf("Unexpected error: %v", err)
			}
			if capturedTarget != tt.expectedTarget {
				t.Errorf("Expected target %q, got %q", tt.expectedTarget, capturedTarget)
			}
			if len(capturedCommand) != len(tt.expectedCommand) {
				t.Fatalf("Expected command %v, got %v", tt.expectedCommand, capturedCommand)
			}
			for i, c := range capturedCommand {
				if c != tt.expectedCommand[i] {
					t.Errorf("Command arg %d: expected %q, got %q", i, tt.expectedCommand[i], c)
				}
			}
		})
	}
}

// TestExecuteCommandWithHEAD tests that execute falls back to HEAD when given 1 arg
func TestExecuteCommandWithHEAD(t *testing.T) {
	// Create a temp dir and set up HEAD
	tempDir, err := os.MkdirTemp("", "vers-execute-head-test-")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	originalDir, err := os.Getwd()
	if err != nil {
		t.Fatalf("Failed to get current directory: %v", err)
	}
	defer os.Chdir(originalDir)

	err = os.Chdir(tempDir)
	if err != nil {
		t.Fatalf("Failed to change to temp directory: %v", err)
	}

	// Set up a HEAD VM
	testVMID := "test-vm-12345678-1234-1234-1234-123456789abc"
	err = utils.SetHead(testVMID)
	if err != nil {
		t.Fatalf("Failed to set HEAD: %v", err)
	}

	// Verify HEAD is set
	head, err := utils.GetCurrentHeadVM()
	if err != nil {
		t.Fatalf("Failed to get HEAD: %v", err)
	}
	if head != testVMID {
		t.Fatalf("Expected HEAD %s, got %s", testVMID, head)
	}

	// Simulate the execute command's arg parsing with 1 arg (command only)
	args := []string{"echo hello"}
	var target string
	var command []string

	if len(args) == 1 {
		target = ""
		command = args
	} else {
		target = args[0]
		command = args[1:]
	}

	// When target is empty, HEAD should be used
	if target != "" {
		t.Fatalf("Expected empty target for HEAD fallback, got %q", target)
	}

	// Simulate what HandleExecute does when target is empty
	headVM, err := utils.GetCurrentHeadVM()
	if err != nil {
		t.Fatalf("Expected HEAD VM to be available: %v", err)
	}
	if headVM != testVMID {
		t.Fatalf("Expected HEAD VM %s, got %s", testVMID, headVM)
	}
	if len(command) != 1 || command[0] != "echo hello" {
		t.Fatalf("Expected command [echo hello], got %v", command)
	}
}

// TestExecuteCommandWithoutHEAD tests that execute fails gracefully when no HEAD is set and 1 arg
func TestExecuteCommandWithoutHEAD(t *testing.T) {
	// Create a temp dir WITHOUT HEAD
	tempDir, err := os.MkdirTemp("", "vers-execute-nohead-test-")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	originalDir, err := os.Getwd()
	if err != nil {
		t.Fatalf("Failed to get current directory: %v", err)
	}
	defer os.Chdir(originalDir)

	err = os.Chdir(tempDir)
	if err != nil {
		t.Fatalf("Failed to change to temp directory: %v", err)
	}

	// Simulate what happens when target is empty and no HEAD is set
	_, err = utils.GetCurrentHeadVM()
	if err == nil {
		t.Fatal("Expected error when no HEAD is set")
	}
}
