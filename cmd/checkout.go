package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

// checkoutCmd represents the checkout command
var checkoutCmd = &cobra.Command{
	Use:   "checkout <branch>",
	Short: "Checkout a branch or commit",
	Long:  `Switch to a different branch or commit in the Vers environment.`,
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		branchName := args[0]
		
		fmt.Printf("Checking out branch/commit: %s\n", branchName)

		// Call the SDK to checkout the branch
		if err := client.CheckoutBranch(branchName); err != nil {
			return fmt.Errorf("checkout failed: %w", err)
		}

		fmt.Printf("Successfully checked out: %s\n", branchName)
		return nil
	},
}

func init() {
	rootCmd.AddCommand(checkoutCmd)

	// No additional flags needed for checkout
} 