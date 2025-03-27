package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

var fromBranch string

// branchCmd represents the branch command
var branchCmd = &cobra.Command{
	Use:   "branch <name>",
	Short: "Create a new branch",
	Long:  `Create a new branch from the current state or a specific branch/commit.`,
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		branchName := args[0]
		
		if fromBranch == "" {
			fmt.Printf("Creating branch '%s' from current state\n", branchName)
		} else {
			fmt.Printf("Creating branch '%s' from '%s'\n", branchName, fromBranch)
		}

		// Call the SDK to create a branch
		if err := client.CreateBranch(branchName, fromBranch); err != nil {
			return fmt.Errorf("branch creation failed: %w", err)
		}

		fmt.Printf("Successfully created branch: %s\n", branchName)
		return nil
	},
}

func init() {
	rootCmd.AddCommand(branchCmd)

	// Define flags for the branch command
	branchCmd.Flags().StringVarP(&fromBranch, "from", "f", "", "Source branch or commit (default: current state)")
} 