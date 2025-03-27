package cmd

import (
	"fmt"

	"github.com/AlecAivazis/survey/v2"
	"github.com/hdresearch/vers-cli/internal/sdk"
	"github.com/spf13/cobra"
)

var token string

// loginCmd represents the login command
var loginCmd = &cobra.Command{
	Use:   "login",
	Short: "Authenticate with the Vers platform",
	Long:  `Login to the Vers platform using your credentials or API token.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		// If token is not provided via flag, prompt for it
		if token == "" {
			prompt := &survey.Password{
				Message: "Enter your API token:",
			}
			err := survey.AskOne(prompt, &token, survey.WithValidator(survey.Required))
			if err != nil {
				return fmt.Errorf("error getting token: %w", err)
			}
		}

		// Call the SDK to handle login
		client = sdk.NewStubClient(nil)
		if err := client.Login(token); err != nil {
			return fmt.Errorf("login failed: %w", err)
		}

		fmt.Println("Successfully logged in to Vers platform")
		return nil
	},
}

func init() {
	rootCmd.AddCommand(loginCmd)

	// Define flags for the login command
	loginCmd.Flags().StringVarP(&token, "token", "t", "", "API token for authentication")
} 