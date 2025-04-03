package cmd

import (
	"github.com/spf13/cobra"
)

// pickCmd represents the pick command
var pickCmd = &cobra.Command{
	Use:   "pick <branch>",
	Short: "Select a branch to keep",
	Long:  `Select a specific branch to keep, discarding others.`,
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		// branchName := args[0]
		
		// // fmt.Printf("Selecting branch to keep: %s\n", branchName)

		// // // Initialize the context for future SDK calls
		// // _ = context.Background()
		
		// // // Call the SDK to pick the branch
		// // // This is a stub implementation - adjust based on actual SDK API
		// // fmt.Println("Keeping selected branch...")
		// // // Example: response, err := client.API.State.Pick(ctx, branchName)

		// // fmt.Printf("Successfully kept branch: %s\n", branchName)
		return nil
	},
}

func init() {
	rootCmd.AddCommand(pickCmd)

	// No additional flags needed for pick
} 