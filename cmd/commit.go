package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

var commitMsg string
var tag string

// commitCmd represents the commit command
var commitCmd = &cobra.Command{
	Use:   "commit",
	Short: "Commit the current state of the environment",
	Long:  `Save the current state of the Vers environment as a commit.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		// // Validate that a commit message is provided
		// if commitMsg == "" {
		// 	return fmt.Errorf("a commit message is required, use -m or --message flag")
		// }

		// fmt.Printf("Creating commit with message: %s\n", commitMsg)
		// if tag != "" {
		// 	fmt.Printf("Tagging commit as: %s\n", tag)
		// }

		// // Initialize the context for future SDK calls
		// _ = context.Background()

		// // Call the SDK to commit the VM state
		// // This is a stub implementation - adjust based on actual SDK API
		// fmt.Println("Creating commit...")
		// // Example: response, err := client.API.State.Commit(ctx, commitMsg, tag)

		// fmt.Println("Successfully committed the current state")
		fmt.Println("Error: Not implemented yet. We will be adding this soon.")
		return nil
	},
}

func init() {
	rootCmd.AddCommand(commitCmd)

	// Define flags for the commit command
	commitCmd.Flags().StringVarP(&commitMsg, "message", "m", "", "Commit message (required)")
	commitCmd.Flags().StringVarP(&tag, "tag", "t", "", "Tag for this commit")

	// Mark message as required
	commitCmd.MarkFlagRequired("message")
}
