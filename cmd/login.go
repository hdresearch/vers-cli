package cmd

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	"github.com/hdresearch/vers-cli/internal/auth"
	"github.com/spf13/cobra"
)

var token string

// loginCmd represents the login command
var loginCmd = &cobra.Command{
	Use:   "login",
	Short: "Authenticate with the Vers platform",
	Long:  `Login to the Vers platform using your credentials or API token.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		if token == "" {
			fmt.Print("Enter your API key: ")
			reader := bufio.NewReader(os.Stdin)
			input, err := reader.ReadString('\n')
			if err != nil {
				return fmt.Errorf("error reading input: %w", err)
			}
			token = strings.TrimSpace(input)
			if token == "" {
				return fmt.Errorf("API key cannot be empty, no changes made")
			}
		}

		err := auth.SaveAPIKey(token)
		if err != nil {
			return fmt.Errorf("error saving API key: %w", err)
		}

		fmt.Println("Successfully authenticated with Vers")
		return nil
	},
}

func init() {
	rootCmd.AddCommand(loginCmd)

	// Define flags for the login command
	loginCmd.Flags().StringVarP(&token, "token", "t", "", "API token for authentication")
}
