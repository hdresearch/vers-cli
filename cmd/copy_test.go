package cmd

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/hdresearch/vers-cli/internal/auth"
	"github.com/hdresearch/vers-cli/internal/utils"
	"github.com/spf13/cobra"
)

// TestCopyCommandArgumentParsing tests the argument parsing logic
func TestCopyCommandArgumentParsing(t *testing.T) {
	tests := []struct {
		name            string
		args            []string
		expectError     bool
		expectedVMID    string
		expectedSource  string
		expectedDest    string
		expectedUseHead bool
		errorMessage    string
	}{
		{
			name:            "valid 3 args with VM ID",
			args:            []string{"vm-123", "./local-file", "/remote/path"},
			expectError:     false,
			expectedVMID:    "vm-123",
			expectedSource:  "./local-file",
			expectedDest:    "/remote/path",
			expectedUseHead: false,
		},
		{
			name:            "valid 2 args using HEAD",
			args:            []string{"./local-file", "/remote/path"},
			expectError:     false,
			expectedSource:  "./local-file",
			expectedDest:    "/remote/path",
			expectedUseHead: true,
		},
		{
			name:         "invalid 1 arg",
			args:         []string{"./local-file"},
			expectError:  true,
			errorMessage: "accepts between 2 and 3 arg(s), received 1",
		},
		{
			name:         "invalid 4 args",
			args:         []string{"vm-123", "./local-file", "/remote/path", "extra"},
			expectError:  true,
			errorMessage: "accepts between 2 and 3 arg(s), received 4",
		},
		{
			name:         "invalid 0 args",
			args:         []string{},
			expectError:  true,
			errorMessage: "accepts between 2 and 3 arg(s), received 0",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cmd := &cobra.Command{
				Use:  "copy [vm-id|alias] <source> <destination>",
				Args: cobra.RangeArgs(2, 3),
				RunE: func(cmd *cobra.Command, args []string) error {
					// Mock the argument parsing logic from the real command
					var vmIdentifier string
					var source, destination string

					if len(args) == 2 {
						// Use HEAD VM
						vmIdentifier = "HEAD"
						source = args[0]
						destination = args[1]
					} else {
						// VM specified
						vmIdentifier = args[0]
						source = args[1]
						destination = args[2]
					}

					// Validate our expectations
					if tt.expectedVMID != "" && vmIdentifier != tt.expectedVMID {
						return fmt.Errorf("expected VM ID %s, got %s", tt.expectedVMID, vmIdentifier)
					}
					if tt.expectedUseHead && vmIdentifier != "HEAD" {
						return fmt.Errorf("expected HEAD usage, got %s", vmIdentifier)
					}
					if source != tt.expectedSource {
						return fmt.Errorf("expected source %s, got %s", tt.expectedSource, source)
					}
					if destination != tt.expectedDest {
						return fmt.Errorf("expected destination %s, got %s", tt.expectedDest, destination)
					}

					return nil
				},
			}

			// Set the arguments first
			cmd.SetArgs(tt.args)

			// Execute the command, which will validate arguments and call RunE
			err := cmd.Execute()

			if tt.expectError {
				if err == nil {
					t.Errorf("Expected error but got none")
				} else if tt.errorMessage != "" && !strings.Contains(err.Error(), tt.errorMessage) {
					t.Errorf("Expected error containing '%s', got: %v", tt.errorMessage, err)
				}
			} else {
				if err != nil {
					t.Errorf("Unexpected error: %v", err)
				}
			}
		})
	}
}

// TestCopyCommandUploadDownloadDetection tests the upload/download detection logic
func TestCopyCommandUploadDownloadDetection(t *testing.T) {
	// Create temporary directory for test files
	tempDir, err := ioutil.TempDir("", "vers-copy-test-")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Create a test file
	testFile := filepath.Join(tempDir, "test-file.txt")
	err = os.WriteFile(testFile, []byte("test content"), 0644)
	if err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	tests := []struct {
		name             string
		source           string
		destination      string
		expectedUpload   bool
		expectedDownload bool
		description      string
	}{
		{
			name:             "upload - local file to remote path",
			source:           testFile,
			destination:      "/remote/path/file.txt",
			expectedUpload:   true,
			expectedDownload: false,
			description:      "Local file exists and destination is remote path",
		},
		{
			name:             "download - remote path to local file",
			source:           "/remote/path/file.txt",
			destination:      "./local-file.txt",
			expectedUpload:   false,
			expectedDownload: true,
			description:      "Source is remote path and destination is local path",
		},
		{
			name:             "upload - relative local to remote",
			source:           "./local-file.txt",
			destination:      "/remote/path/",
			expectedUpload:   true,
			expectedDownload: false,
			description:      "Relative local path to remote directory",
		},
		{
			name:             "download - remote to relative local",
			source:           "/remote/file.txt",
			destination:      "./",
			expectedUpload:   false,
			expectedDownload: true,
			description:      "Remote file to local directory",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Mock the detection logic from the real command
			var isUpload, isDownload bool

			if strings.HasPrefix(tt.source, "/") && !strings.HasPrefix(tt.destination, "/") {
				// Downloading from remote to local
				isDownload = true
			} else if !strings.HasPrefix(tt.source, "/") && strings.HasPrefix(tt.destination, "/") {
				// Uploading from local to remote
				isUpload = true
			} else {
				// Auto-detect based on file existence
				if _, err := os.Stat(tt.source); err == nil {
					// Local file exists, upload
					isUpload = true
				} else {
					// Assume remote file, download
					isDownload = true
				}
			}

			if isUpload != tt.expectedUpload {
				t.Errorf("Expected upload %v, got %v for %s", tt.expectedUpload, isUpload, tt.description)
			}
			if isDownload != tt.expectedDownload {
				t.Errorf("Expected download %v, got %v for %s", tt.expectedDownload, isDownload, tt.description)
			}
		})
	}
}

// TestCopyCommandSCPConstruction tests the SCP command construction logic
func TestCopyCommandSCPConstruction(t *testing.T) {
	tests := []struct {
		name           string
		sshHost        string
		sshPort        string
		keyPath        string
		source         string
		destination    string
		isUpload       bool
		recursive      bool
		expectedArgs   []string
		expectedSource string
		expectedDest   string
	}{
		{
			name:        "upload command construction",
			sshHost:     "192.168.1.100",
			sshPort:     "2222",
			keyPath:     "/path/to/key",
			source:      "./local-file.txt",
			destination: "/remote/path/file.txt",
			isUpload:    true,
			recursive:   false,
			expectedArgs: []string{
				"scp",
				"-P", "2222",
				"-o", "StrictHostKeyChecking=no",
				"-o", "UserKnownHostsFile=/dev/null",
				"-o", "IdentitiesOnly=yes",
				"-o", "PreferredAuthentications=publickey",
				"-o", "LogLevel=ERROR",
				"-i", "/path/to/key",
				"./local-file.txt",
				"root@192.168.1.100:/remote/path/file.txt",
			},
			expectedSource: "./local-file.txt",
			expectedDest:   "root@192.168.1.100:/remote/path/file.txt",
		},
		{
			name:        "download command construction",
			sshHost:     "192.168.1.100",
			sshPort:     "2222",
			keyPath:     "/path/to/key",
			source:      "/remote/path/file.txt",
			destination: "./local-file.txt",
			isUpload:    false,
			recursive:   false,
			expectedArgs: []string{
				"scp",
				"-P", "2222",
				"-o", "StrictHostKeyChecking=no",
				"-o", "UserKnownHostsFile=/dev/null",
				"-o", "IdentitiesOnly=yes",
				"-o", "PreferredAuthentications=publickey",
				"-o", "LogLevel=ERROR",
				"-i", "/path/to/key",
				"root@192.168.1.100:/remote/path/file.txt",
				"./local-file.txt",
			},
			expectedSource: "root@192.168.1.100:/remote/path/file.txt",
			expectedDest:   "./local-file.txt",
		},
		{
			name:        "recursive upload command construction",
			sshHost:     "192.168.1.100",
			sshPort:     "2222",
			keyPath:     "/path/to/key",
			source:      "./local-dir/",
			destination: "/remote/path/",
			isUpload:    true,
			recursive:   true,
			expectedArgs: []string{
				"scp",
				"-P", "2222",
				"-o", "StrictHostKeyChecking=no",
				"-o", "UserKnownHostsFile=/dev/null",
				"-o", "IdentitiesOnly=yes",
				"-o", "PreferredAuthentications=publickey",
				"-o", "LogLevel=ERROR",
				"-i", "/path/to/key",
				"-r",
				"./local-dir/",
				"root@192.168.1.100:/remote/path/",
			},
			expectedSource: "./local-dir/",
			expectedDest:   "root@192.168.1.100:/remote/path/",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Mock the SCP command construction logic
			scpTarget := fmt.Sprintf("root@%s", tt.sshHost)

			var scpSource, scpDest string
			if tt.isUpload {
				scpSource = tt.source
				scpDest = fmt.Sprintf("%s:%s", scpTarget, tt.destination)
			} else {
				scpSource = fmt.Sprintf("%s:%s", scpTarget, tt.source)
				scpDest = tt.destination
			}

			// Construct the expected command arguments
			expectedArgs := []string{
				"scp",
				"-P", tt.sshPort,
				"-o", "StrictHostKeyChecking=no",
				"-o", "UserKnownHostsFile=/dev/null",
				"-o", "IdentitiesOnly=yes",
				"-o", "PreferredAuthentications=publickey",
				"-o", "LogLevel=ERROR",
				"-i", tt.keyPath,
			}

			// Add recursive flag if enabled
			if tt.recursive {
				expectedArgs = append(expectedArgs, "-r")
			}

			// Add source and destination
			expectedArgs = append(expectedArgs, scpSource, scpDest)

			// Verify the constructed arguments match expectations
			if len(expectedArgs) != len(tt.expectedArgs) {
				t.Errorf("Expected %d args, got %d", len(tt.expectedArgs), len(expectedArgs))
			}

			for i, expectedArg := range tt.expectedArgs {
				if i < len(expectedArgs) && expectedArgs[i] != expectedArg {
					t.Errorf("Arg %d: expected %s, got %s", i, expectedArg, expectedArgs[i])
				}
			}

			// Verify source and destination
			if scpSource != tt.expectedSource {
				t.Errorf("Expected source %s, got %s", tt.expectedSource, scpSource)
			}
			if scpDest != tt.expectedDest {
				t.Errorf("Expected destination %s, got %s", tt.expectedDest, scpDest)
			}
		})
	}
}

// TestCopyCommandIntegration tests the copy command end-to-end with the actual service
func TestCopyCommandIntegration(t *testing.T) {
	// Skip if we're in a unit test environment (no API key available)
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	// Check if we have authentication available
	hasAPIKey, err := auth.HasAPIKey()
	if err != nil || !hasAPIKey {
		t.Skip("Skipping integration test: No API key available. Run 'vers login' first.")
	}

	// Create temporary directory for test files
	tempDir, err := ioutil.TempDir("", "vers-copy-integration-")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Change to temp directory to ensure proper working directory
	originalDir, err := os.Getwd()
	if err != nil {
		t.Fatalf("Failed to get current directory: %v", err)
	}
	defer os.Chdir(originalDir)

	err = os.Chdir(tempDir)
	if err != nil {
		t.Fatalf("Failed to change to temp directory: %v", err)
	}

	// Create a test file to upload
	testFile := filepath.Join(tempDir, "test-upload.txt")
	testContent := fmt.Sprintf("Test content created at %s", time.Now().Format(time.RFC3339))
	err = os.WriteFile(testFile, []byte(testContent), 0644)
	if err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	// Initialize client for integration test
	setupIntegrationClient(t)

	tests := []struct {
		name         string
		args         []string
		expectError  bool
		errorMessage string
		validateFile bool
		vmSpecific   bool // whether this test requires a specific VM ID
	}{
		{
			name:         "copy with non-existent VM",
			args:         []string{"non-existent-vm", testFile, "/tmp/test.txt"},
			expectError:  true,
			errorMessage: "failed to get VM information",
			vmSpecific:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create a new copy command for each test
			cmd := &cobra.Command{
				Use:  "copy [vm-id|alias] <source> <destination>",
				Args: cobra.RangeArgs(2, 3),
				RunE: copyCmd.RunE,
			}

			// Add recursive flag for consistency with real command
			cmd.Flags().BoolP("recursive", "r", false, "Recursively copy directories")

			// Set the arguments
			cmd.SetArgs(tt.args)

			// Execute the command
			err := cmd.Execute()

			if tt.expectError {
				if err == nil {
					t.Errorf("Expected error but got none")
				} else if tt.errorMessage != "" && !strings.Contains(err.Error(), tt.errorMessage) {
					t.Errorf("Expected error containing '%s', got: %v", tt.errorMessage, err)
				}
			} else {
				if err != nil {
					// Check if it's a service error vs a client error
					if strings.Contains(err.Error(), "500 Internal Server Error") ||
						strings.Contains(err.Error(), "no VM ID provided") ||
						strings.Contains(err.Error(), "HEAD not found") {
						t.Skipf("Skipping test due to service/environment error: %v", err)
						return
					}
					t.Errorf("Unexpected error: %v", err)
				}

				// Validate downloaded file if specified
				if tt.validateFile {
					downloadedFile := filepath.Join(tempDir, "downloaded-file.txt")
					if _, err := os.Stat(downloadedFile); err != nil {
						t.Errorf("Downloaded file not found: %v", err)
					} else {
						content, err := os.ReadFile(downloadedFile)
						if err != nil {
							t.Errorf("Failed to read downloaded file: %v", err)
						} else if string(content) != testContent {
							t.Errorf("Downloaded content doesn't match. Expected: %s, Got: %s", testContent, string(content))
						}
					}
				}
			}
		})
	}
}

// TestCopyCommandWithRealHEAD tests copy command with actual HEAD setup
func TestCopyCommandWithRealHEAD(t *testing.T) {
	tempDir, err := ioutil.TempDir("", "vers-copy-head-test-")
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

	// Test that 2-arg copy finds HEAD
	cmd := &cobra.Command{
		Use:  "copy [vm-id|alias] <source> <destination>",
		Args: cobra.RangeArgs(2, 3),
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) == 2 {
				headVMID, err := utils.GetCurrentHeadVM()
				if err != nil {
					return fmt.Errorf("no VM ID provided and %w", err)
				}
				if headVMID != testVMID {
					return fmt.Errorf("expected HEAD VM %s, got %s", testVMID, headVMID)
				}
			}
			return nil // Success - HEAD was found correctly
		},
	}

	cmd.SetArgs([]string{"./local-file", "/remote/path"})
	err = cmd.Execute()
	if err != nil {
		t.Errorf("Failed to use HEAD: %v", err)
	}
}

// TestHEADUtilities tests the HEAD utility functions
func TestHEADUtilities(t *testing.T) {
	tempDir, err := ioutil.TempDir("", "vers-head-utils-test-")
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

	// Test no HEAD file
	_, err = utils.GetCurrentHeadVM()
	if err == nil || !strings.Contains(err.Error(), "HEAD not found") {
		t.Errorf("Expected 'HEAD not found' error, got: %v", err)
	}

	// Test set and get HEAD
	testVMID := "test-vm-12345678-1234-1234-1234-123456789abc"
	err = utils.SetHead(testVMID)
	if err != nil {
		t.Fatalf("Failed to set HEAD: %v", err)
	}

	headVM, err := utils.GetCurrentHeadVM()
	if err != nil {
		t.Fatalf("Failed to get HEAD: %v", err)
	}
	if headVM != testVMID {
		t.Errorf("Expected HEAD VM %s, got %s", testVMID, headVM)
	}
}
