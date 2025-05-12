package cmd

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/hdresearch/vers-cli/internal/auth"
	"github.com/spf13/cobra"
)

// getCurrentHeadVM returns the VM ID from the current HEAD
func getCurrentHeadVM() (string, error) {
	versDir := ".vers"
	headFile := filepath.Join(versDir, "HEAD")

	// Check if .vers directory and HEAD file exist
	if _, err := os.Stat(headFile); os.IsNotExist(err) {
		return "", fmt.Errorf("HEAD not found. Run 'vers init' first")
	}

	// Read HEAD file
	headData, err := os.ReadFile(headFile)
	if err != nil {
		return "", fmt.Errorf("error reading HEAD: %w", err)
	}

	// Parse the HEAD content
	headContent := string(strings.TrimSpace(string(headData)))
	var vmID string

	// Check if HEAD is a symbolic ref or direct ref
	if strings.HasPrefix(headContent, "ref: ") {
		// It's a symbolic ref, extract the path
		refPath := strings.TrimPrefix(headContent, "ref: ")

		// Read the actual reference file
		refFile := filepath.Join(versDir, refPath)
		refData, err := os.ReadFile(refFile)
		if err != nil {
			return "", fmt.Errorf("error reading reference '%s': %w", refPath, err)
		}

		// Get the VM ID from the reference file
		vmID = string(strings.TrimSpace(string(refData)))
	} else {
		// HEAD directly contains a VM ID
		vmID = headContent
	}

	if vmID == "" {
		return "", fmt.Errorf("could not determine current VM ID from HEAD")
	}

	return vmID, nil
}

// executeCmd represents the execute command
var executeCmd = &cobra.Command{
	Use:   "execute [vm_id] <command> [args...]",
	Short: "Run a command on a specific VM",
	Long:  `Execute a command within the Vers environment on the specified VM. If no VM ID is provided, uses the current HEAD.`,
	Args:  cobra.MinimumNArgs(1), // Require at least command
	RunE: func(cmd *cobra.Command, args []string) error {
		var vmID string
		var commandArgs []string
		var commandStr string
		s := NewStatusStyles()

		// Check if first arg is a VM ID or a command
		if len(args) > 1 {
			vmID = args[0]
			commandArgs = args[1:]
		} else {
			// First arg doesn't look like a VM ID or only one arg, use HEAD
			var err error
			vmID, err = getCurrentHeadVM()
			if err != nil {
				return fmt.Errorf(s.NoData.Render("no VM ID provided and %v"), err)
			}
			fmt.Printf(s.HeadStatus.Render("Using current HEAD VM: "+vmID) + "\n")
			commandArgs = args
		}

		// Join the command arguments
		commandStr = strings.Join(commandArgs, " ")

		// Initialize SDK client and context
		baseCtx := context.Background()
		apiCtx, cancel := context.WithTimeout(baseCtx, 30*time.Second)
		defer cancel()

		response, err := client.API.Vm.Get(apiCtx, vmID)
		if err != nil {
			return fmt.Errorf(s.NoData.Render("failed to get VM information: %v"), err)
		}
		vm := response.Data

		if vm.State != "Running" {
			return fmt.Errorf(s.NoData.Render("VM is not running (current state: %s)"), vm.State)
		}

		if vm.NetworkInfo.SSHPort == 0 {
			return fmt.Errorf("%s", s.NoData.Render("VM does not have SSH port information available"))
		}

		// Determine the path for storing the SSH key
		keyPath := getSSHKeyPath(vmID)

		// Check if SSH key already exists
		keyExists := false
		if _, err := os.Stat(keyPath); err == nil {
			keyExists = true
		}

		// If key doesn't exist, fetch it and save it
		if !keyExists {
			// Create the keys directory if it doesn't exist
			keysDir := filepath.Dir(keyPath)
			if err := os.MkdirAll(keysDir, 0755); err != nil {
				return fmt.Errorf(s.NoData.Render("failed to create keys directory: %v"), err)
			}

			// Get SSH key using SDK
			response, err := client.API.Vm.GetSSHKey(apiCtx, vmID)
			if err != nil {
				return fmt.Errorf(s.NoData.Render("failed to get SSH key: %v"), err)
			}
			sshKeyBytes := response.Data

			// Write key to file
			if err := os.WriteFile(keyPath, []byte(sshKeyBytes), 0600); err != nil {
				return fmt.Errorf(s.NoData.Render("failed to write key file: %v"), err)
			}

		}

		hostIP := auth.GetVersUrl()

		// // Debug info about connection
		// fmt.Printf(s.HeadStatus.Render("Executing command via SSH on %s (VM %s)\n"), hostIP, vmID)

		// Create the SSH command with the provided command string
		sshCmd := exec.Command("ssh",
			fmt.Sprintf("root@%s", hostIP),
			"-p", fmt.Sprintf("%d", vm.NetworkInfo.SSHPort),
			"-o", "StrictHostKeyChecking=no",
			"-o", "UserKnownHostsFile=/dev/null", // Avoid host key prompts
			"-o", "IdentitiesOnly=yes", // Only use the specified identity file
			"-o", "PreferredAuthentications=publickey", // Only attempt public key authentication
			"-o", "LogLevel=ERROR", // Add this line to suppress warnings
			"-i", keyPath, // Use the persistent key file
			commandStr) // Add the command to execute

		// Connect command output to current terminal
		sshCmd.Stdout = os.Stdout
		sshCmd.Stderr = os.Stderr

		// Execute the command
		err = sshCmd.Run()
		if err != nil {
			if exitErr, ok := err.(*exec.ExitError); ok {
				// If the command ran but returned non-zero, return the exit code
				return fmt.Errorf(s.NoData.Render("command exited with code %d"), exitErr.ExitCode())
			}
			return fmt.Errorf(s.NoData.Render("failed to run SSH command: %v"), err)
		}

		return nil
	},
}

func init() {
	rootCmd.AddCommand(executeCmd)
	executeCmd.Flags().String("host", "", "Specify the host IP to connect to (overrides default)")
}
