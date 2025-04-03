package cmd

import (
	"context"
	"fmt"
	"strings"
	"time"

	vers "github.com/hdresearch/vers-sdk-go"
	"github.com/spf13/cobra"
)

// executeCmd represents the execute command
var executeCmd = &cobra.Command{
	Use:   "execute <vm_id> <command> [args...]",
	Short: "Run a command on a specific VM",
	Long:  `Execute a command within the Vers environment on the specified VM.`,
	Args:  cobra.MinimumNArgs(2), // Require at least vm_id and command
	RunE: func(cmd *cobra.Command, args []string) error {
		// The first argument is the vm_id
		vmID := args[0]
		// The rest of the arguments form the command
		commandArgs := args[1:]
		commandStr := strings.Join(commandArgs, " ")

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