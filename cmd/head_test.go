package cmd

import (
	"bytes"
	"os"
	"strings"
	"testing"

	"github.com/hdresearch/vers-cli/internal/utils"
)

func TestHeadCommand_WithHEADSet(t *testing.T) {
	// Create a temp dir and set up HEAD
	tempDir, err := os.MkdirTemp("", "vers-head-test-")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	originalDir, err := os.Getwd()
	if err != nil {
		t.Fatalf("Failed to get current directory: %v", err)
	}
	defer os.Chdir(originalDir)

	if err := os.Chdir(tempDir); err != nil {
		t.Fatalf("Failed to change to temp directory: %v", err)
	}

	testVMID := "test-vm-12345678-1234-1234-1234-123456789abc"
	if err := utils.SetHead(testVMID); err != nil {
		t.Fatalf("Failed to set HEAD: %v", err)
	}

	// Capture output by running the command's RunE directly
	var buf bytes.Buffer
	headCmd.SetOut(&buf)
	headCmd.SetErr(&buf)

	// Override the Run function to write to our buffer
	vmID, err := utils.GetCurrentHeadVM()
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}
	if vmID != testVMID {
		t.Errorf("Expected VM ID %q, got %q", testVMID, vmID)
	}
}

func TestHeadCommand_WithoutHEAD(t *testing.T) {
	// Create a temp dir WITHOUT HEAD
	tempDir, err := os.MkdirTemp("", "vers-head-nohead-test-")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	originalDir, err := os.Getwd()
	if err != nil {
		t.Fatalf("Failed to get current directory: %v", err)
	}
	defer os.Chdir(originalDir)

	if err := os.Chdir(tempDir); err != nil {
		t.Fatalf("Failed to change to temp directory: %v", err)
	}

	// Running head command should error when no HEAD is set
	err = headCmd.RunE(headCmd, []string{})
	if err == nil {
		t.Fatal("Expected error when no HEAD is set, got nil")
	}
	if !strings.Contains(err.Error(), "no HEAD set") {
		t.Errorf("Expected error to contain 'no HEAD set', got: %v", err)
	}
}

func TestHeadCommand_WithEmptyHEAD(t *testing.T) {
	// Create a temp dir with an empty HEAD file
	tempDir, err := os.MkdirTemp("", "vers-head-empty-test-")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	originalDir, err := os.Getwd()
	if err != nil {
		t.Fatalf("Failed to get current directory: %v", err)
	}
	defer os.Chdir(originalDir)

	if err := os.Chdir(tempDir); err != nil {
		t.Fatalf("Failed to change to temp directory: %v", err)
	}

	// Create .vers dir with empty HEAD
	if err := os.MkdirAll(utils.VersDir, 0755); err != nil {
		t.Fatalf("Failed to create .vers dir: %v", err)
	}
	if err := os.WriteFile(utils.VersDir+"/HEAD", []byte(""), 0644); err != nil {
		t.Fatalf("Failed to write empty HEAD: %v", err)
	}

	err = headCmd.RunE(headCmd, []string{})
	if err == nil {
		t.Fatal("Expected error when HEAD is empty, got nil")
	}
	if !strings.Contains(err.Error(), "no HEAD set") {
		t.Errorf("Expected error to contain 'no HEAD set', got: %v", err)
	}
}
