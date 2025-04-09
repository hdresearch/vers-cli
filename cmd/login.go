package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

var token string

// loginCmd represents the login command
var loginCmd = &cobra.Command{
	Use:   "login",
	Short: "Authenticate with the Vers platform",
	Long:  `Login to the Vers platform using your credentials or API token.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		// // If token is not provided via flag, prompt for it
		// if token == "" {
		// 	prompt := &survey.Password{
		// 		Message: "Enter your API token:",
		// 	}
		// 	err := survey.AskOne(prompt, &token, survey.WithValidator(survey.Required))
		// 	if err != nil {
		// 		return fmt.Errorf("error getting token: %w", err)
		// 	}
		// }

		// // Call the SDK to handle login
		// client = vers.NewClient(
		// 	option.WithAPIKey(token),
		// )

		// // Verify the token works by making a simple API call
		// fmt.Println("Verifying API token...")
		// // You would typically make a simple API call here to verify the token
		// // For example: _, err := client.API.SomeSimpleEndpoint.Get(context.TODO())

		// fmt.Println("Successfully logged in to Vers platform")

		// // Save the token for future use
		// // This would typically involve storing the token in a secure location
		// // like the system keychain or a config file with appropriate permissions
		fmt.Println("Error: Not implemented yet. We will be adding this soon.")
		return nil
	},
}

func init() {
	rootCmd.AddCommand(loginCmd)

	// Define flags for the login command
	loginCmd.Flags().StringVarP(&token, "token", "t", "", "API token for authentication")
}
