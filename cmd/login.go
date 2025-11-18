package cmd

import (
	"context"
	"fmt"
	"strings"
	"syscall"
	"time"

	"github.com/hdresearch/vers-cli/internal/auth"
	vers "github.com/hdresearch/vers-sdk-go"
	"github.com/hdresearch/vers-sdk-go/option"
	"github.com/spf13/cobra"
	"golang.org/x/term"
)

var token string

// validateAPIKey validates the API key by attempting to list VMs
func validateAPIKey(apiKey string) error {
	// Get client options
	clientOptions, err := auth.GetClientOptions()
	if err != nil {
		return fmt.Errorf("error getting client options: %w", err)
	}

	// Add the API key to the options
	clientOptions = append(clientOptions, option.WithAPIKey(apiKey))

	// Create a client with the provided API key
	client := vers.NewClient(clientOptions...)

	// Try to list VMs as a validation check
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	_, err = client.Vm.List(ctx)
	if err != nil {
		// Check if it's an authentication/authorization error
		errStr := err.Error()
		errStrLower := strings.ToLower(errStr)
		if strings.Contains(errStr, "401") || strings.Contains(errStr, "403") ||
			strings.Contains(errStrLower, "unauthorized") || strings.Contains(errStrLower, "forbidden") {
			return fmt.Errorf("invalid API key - please check your key and try again")
		}
		// Other errors might be network issues, etc.
		return fmt.Errorf("could not validate API key: %w", err)
	}

	// Key validated successfully
	fmt.Println("API key validated successfully")
	return nil
}

// secureReadAPIKey reads the API key from stdin without echoing it to the terminal
func secureReadAPIKey() (string, error) {
	fmt.Print("Enter your API key (input will be hidden): ")

	// Read password without echoing
	bytePassword, err := term.ReadPassword(int(syscall.Stdin))
	if err != nil {
		return "", fmt.Errorf("error reading API key: %w", err)
	}

	// Print a newline since ReadPassword doesn't echo one
	fmt.Println()

	apiKey := strings.TrimSpace(string(bytePassword))
	if apiKey == "" {
		return "", fmt.Errorf("API key cannot be empty")
	}

	return apiKey, nil
}

// loginCmd represents the login command
var loginCmd = &cobra.Command{
	Use:   "login",
	Short: "Authenticate with the Vers platform",
	Long: `Login to the Vers platform using your API token.

When you run this command without the --token flag, you'll be prompted
to enter your API key securely (input will be hidden for security).

You can get your API key from: https://vers.sh/dashboard`,
	RunE: func(cmd *cobra.Command, args []string) error {
		if token == "" {
			var err error
			token, err = secureReadAPIKey()
			if err != nil {
				return err
			}
		}

		// Validate the API key - validation must succeed to continue
		fmt.Println("Validating API key...")
		err := validateAPIKey(token)
		if err != nil {
			return err // Stop here if validation fails
		}

		// Save the API key only if validation succeeded
		err = auth.SaveAPIKey(token)
		if err != nil {
			return fmt.Errorf("error saving API key: %w", err)
		}

		fmt.Println("Successfully authenticated with Vers")
		return nil
	},
}

func init() {
	rootCmd.AddCommand(loginCmd)
	loginCmd.Flags().StringVarP(&token, "token", "t", "", "API token for authentication")
}
