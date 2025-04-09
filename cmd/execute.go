package cmd

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	vers "github.com/hdresearch/vers-sdk-go"
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
	headContent := string(bytes.TrimSpace(headData))
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
		vmID = string(bytes.TrimSpace(refData))
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

		// Check if first arg is a VM ID or a command
		if len(args) >= 1 && strings.HasPrefix(args[0], "vm-") {
			// First arg looks like a VM ID, use it
			vmID = args[0]
			commandArgs = args[1:]
		} else {
			// First arg doesn't look like a VM ID or only one arg, use HEAD
			var err error
			vmID, err = getCurrentHeadVM()
			if err != nil {
				return fmt.Errorf("no VM ID provided and %w", err)
			}
			fmt.Printf("Using current HEAD VM: %s\n", vmID)
			commandArgs = args
		}

		// Join the command arguments
		commandStr = strings.Join(commandArgs, " ")

		fmt.Printf("Running command on VM '%s': %s\n", vmID, commandStr)

		// Initialize SDK client and context
		baseCtx := context.Background()
		client := vers.NewClient()
		apiCtx, cancel := context.WithTimeout(baseCtx, 60*time.Second)
		defer cancel()

		// Prepare parameters for the Execute API call
		executeParams := vers.APIVmExecuteParams{
			Command: vers.F(commandStr),
		}

		// Call the SDK to run the command
		fmt.Println("Executing command via Vers SDK...")
		executeResult, err := client.API.Vm.Execute(apiCtx, vmID, executeParams)
		if err != nil {
			return fmt.Errorf("failed to execute command on vm '%s': %w", vmID, err)
		}

		// Handle the response (adjust based on actual APIVmExecuteResult structure)
		fmt.Printf("Command executed successfully on VM '%s'.\n", vmID)
		// Example of how you might display output if the SDK provides it:
		if executeResult != nil {
			if executeResult.CommandResult.Stdout != "" {
				fmt.Println("Output:")
				fmt.Println(executeResult.CommandResult.Stdout)
			}
			if executeResult.CommandResult.Stderr != "" {
				fmt.Println("Error:")
				fmt.Println(executeResult.CommandResult.Stderr)
			}
		} else {
			fmt.Println("No output received or output field not available in response.")
		}

		return nil
	},
}

func init() {
	rootCmd.AddCommand(executeCmd)
}
