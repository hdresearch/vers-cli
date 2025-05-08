package cmd

import (
	"context"
	"fmt"
	"strings"
	"time"

	vers "github.com/hdresearch/vers-sdk-go"
	"github.com/spf13/cobra"
)

// envCmd represents the env command
var envCmd = &cobra.Command{
	Use:   "env [vm_id] KEY=VALUE",
	Short: "Add environment variable to a VM",
	Long:  `Adds an environment variable to the .bashrc file of the specified VM.`,
	Args:  cobra.MinimumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		var vmID string
		var envPair string

		// Check if first arg is a VM ID or an env pair
		if len(args) >= 2 && strings.HasPrefix(args[0], "vm-") {
			// First arg looks like a VM ID, use it
			vmID = args[0]
			envPair = args[1]
		} else {
			// First arg doesn't look like a VM ID, use HEAD
			var err error
			vmID, err = getCurrentHeadVM()
			if err != nil {
				return fmt.Errorf("no VM ID provided and %w", err)
			}
			fmt.Printf("Using current HEAD VM: %s\n", vmID)
			envPair = args[0]
		}

		// Validate env var format (KEY=VALUE)
		if !strings.Contains(envPair, "=") {
			return fmt.Errorf("environment variable must be in format KEY=VALUE")
		}

		parts := strings.SplitN(envPair, "=", 2)
		key := parts[0]
		value := parts[1]

		// Create command to append to .bashrc
		appendCommand := fmt.Sprintf("echo 'export %s=\"%s\"' >> ~/.bashrc", key, value)

		fmt.Printf("Adding environment variable to VM '%s': %s=%s\n", vmID, key, value)

		// Initialize SDK client and context
		baseCtx := context.Background()
		client := vers.NewClient()
		apiCtx, cancel := context.WithTimeout(baseCtx, 30*time.Second)
		defer cancel()

		// Prepare parameters for the Execute API call
		executeParams := vers.APIVmExecuteParams{
			Command: vers.F(appendCommand),
		}

		// Call the SDK to run the command
		executeResult, err := client.API.Vm.Execute(apiCtx, vmID, executeParams)
		if err != nil {
			return fmt.Errorf("failed to add environment variable to VM '%s': %w", vmID, err)
		}

		// Handle the response
		if executeResult != nil && executeResult.CommandResult.ExitCode == 0 {
			fmt.Printf("Successfully added environment variable %s to VM '%s'\n", key, vmID)

			// Source the .bashrc file to make the change take effect immediately
			sourceCommand := "source ~/.bashrc"
			_, err := client.API.Vm.Execute(apiCtx, vmID, vers.APIVmExecuteParams{
				Command: vers.F(sourceCommand),
			})
			if err != nil {
				fmt.Printf("Note: Variable added but couldn't source .bashrc: %v\n", err)
			}
		} else {
			return fmt.Errorf("failed to add environment variable, exit code: %d", executeResult.CommandResult.ExitCode)
		}

		return nil
	},
}

func init() {
	rootCmd.AddCommand(envCmd)
}
