package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

// pickCmd represents the pick command
var pickCmd = &cobra.Command{
	Use:   "pick <branch>",
	Short: "Select a branch to keep",
	Long:  `Select a specific branch to keep, discarding others.`,
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		branchName := args[0]
		
		fmt.Printf("Selecting branch to keep: %s\n", branchName)

		// Call the SDK to pick the branch
		if err := client.PickBranch(branchName); err != nil {
			return fmt.Errorf("pick operation failed: %w", err)
		}

		fmt.Printf("Successfully kept branch: %s\n", branchName)
		return nil
	},
}

func init() {
	rootCmd.AddCommand(pickCmd)

	// No additional flags needed for pick
} 