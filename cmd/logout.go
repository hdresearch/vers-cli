package cmd

import (
	"fmt"

	"github.com/hdresearch/vers-cli/internal/auth"
	"github.com/spf13/cobra"
)

// logoutCmd represents the logout command
var logoutCmd = &cobra.Command{
	Use:   "logout",
	Short: "Log out from the Vers platform",
	Long:  `Log out from the Vers platform by removing your stored API key.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		// Check if an API key is present
		hasKey, err := auth.HasAPIKey()
		if err != nil {
			return fmt.Errorf("error checking for API key: %w", err)
		}
		
		if !hasKey {
			fmt.Println("You are not currently logged in")
			return nil
		}
		
		// Clear the API key by saving an empty string
		err = auth.SaveAPIKey("")
		if err != nil {
			return fmt.Errorf("error removing API key: %w", err)
		}
		
		fmt.Println("Successfully logged out from Vers")
		return nil
	},
}

func init() {
	rootCmd.AddCommand(logoutCmd)
} 