package cmd

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"syscall"

	"github.com/hdresearch/vers-cli/internal/auth"
	"github.com/spf13/cobra"
	"golang.org/x/term"
)

var token string

// validateAPIKey validates the API key against the validation endpoint
func validateAPIKey(apiKey string) error {
	baseURL, err := auth.GetVersUrl()
	if err != nil {
		return fmt.Errorf("error getting API URL: %w", err)
	}
	validateURL := baseURL + "/api/validate"

	payload := map[string]string{
		"api_key": apiKey,
	}

	jsonData, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("error preparing validation request: %w", err)
	}

	req, err := http.NewRequest("POST", validateURL, bytes.NewBuffer(jsonData))
	if err != nil {
		return fmt.Errorf("error creating validation request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+apiKey)

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("could not validate API key: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == 401 {
		return fmt.Errorf("invalid API key - please check your key and try again")
	}

	if resp.StatusCode != 200 {
		return fmt.Errorf("validation failed with status %d - please try again", resp.StatusCode)
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
