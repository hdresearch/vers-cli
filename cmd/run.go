package cmd

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"
)

var machineName string

// runCmd represents the run command
var runCmd = &cobra.Command{
	Use:   "run <command> [args...]",
	Short: "Run a command in the environment",
	Long:  `Execute a command within the Vers environment.`,
	Args:  cobra.MinimumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		// If no machine name is provided, use "default"
		if machineName == "" {
			machineName = "default"
		}
		
		commandStr := strings.Join(args, " ")
		fmt.Printf("Running command on %s: %s\n", machineName, commandStr)

		// Call the SDK to run the command
		if err := client.RunCommand(machineName, args); err != nil {
			return fmt.Errorf("command execution failed: %w", err)
		}

		return nil
	},
}

func init() {
	rootCmd.AddCommand(runCmd)

	// Define flags for the run command
	runCmd.Flags().StringVarP(&machineName, "machine", "m", "", "Target machine to run the command on (default: \"default\")")
} 