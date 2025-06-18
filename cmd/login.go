package cmd

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strings"

	"github.com/hdresearch/vers-cli/internal/auth"
	"github.com/spf13/cobra"
)

var token string

// validateAPIKey validates the API key against vers-lb
// TODO: Remove backward compatibility after migration period (target: later this week should be fine honestly, just don't want to spring this out of nowhere)
func validateAPIKey(apiKey string) error {
	baseURL, err := auth.GetVersUrl()
	if err != nil {
		return fmt.Errorf("error getting API URL: %w", err)
	}
	validateURL := baseURL + "/keys/validate"

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
		// BACKWARD COMPATIBILITY: If validation fails due to network/server issues,
		// allow the key to be saved anyway (old behavior)
		// TODO: Remove this fallback after migration period
		fmt.Printf("Warning: Could not validate API key against server (%v), but saving anyway for backward compatibility\n", err)
		return nil
	}
	defer resp.Body.Close()

	if resp.StatusCode == 401 {
		// BACKWARD COMPATIBILITY: If the key is invalid on the new system,
		// still allow it to be saved (it might be an old-format key)
		// TODO: Remove this fallback after migration period
		fmt.Println("Warning: API key not recognized by new validation system, but saving anyway for backward compatibility")
		return nil
	}

	if resp.StatusCode != 200 {
		// BACKWARD COMPATIBILITY: For other errors, still allow saving
		// TODO: Remove this fallback after migration period
		fmt.Printf("Warning: Validation returned status %d, but saving anyway for backward compatibility\n", resp.StatusCode)
		return nil
	}

	// Key validated successfully against new system
	fmt.Println("API key validated successfully")
	return nil
}

// loginCmd represents the login command
var loginCmd = &cobra.Command{
	Use:   "login",
	Short: "Authenticate with the Vers platform",
	Long:  `Login to the Vers platform using your API token.`,
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

		// Attempt to validate the API key against vers-lb
		// This is non-blocking for backward compatibility
		err := validateAPIKey(token)
		if err != nil {
			// This should never happen with current implementation, but just in case
			return fmt.Errorf("unexpected validation error: %w", err)
		}

		// Save the API key (validated or not, for backward compatibility)
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
