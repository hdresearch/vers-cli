package cmd

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"strings"
	"syscall"
	"time"

	"github.com/hdresearch/vers-cli/internal/auth"
	vers "github.com/hdresearch/vers-sdk-go"
	"github.com/hdresearch/vers-sdk-go/option"
	"github.com/spf13/cobra"
	"golang.org/x/term"
)

var (
	token    string
	loginGit bool
)

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

// loginWithGit authenticates using the Shell Auth flow with git email + SSH key.
func loginWithGit() (retErr error) {
	if telemetryClient != nil {
		telemetryClient.BeginAuthenticationFlow()
	}
	defer func() {
		if retErr != nil {
			trackAuthFailure("signin_completed", "login", "shell_auth", retErr)
		}
	}()

	// Step 1: Get git email
	fmt.Print("Looking up git email... ")
	email, err := auth.GetGitEmail()
	if err != nil {
		fmt.Println("✗")
		return err
	}
	fmt.Println(email)

	// Step 2: Find SSH public key
	fmt.Print("Looking up SSH public key... ")
	sshPubKey, _, err := auth.FindSSHPublicKey()
	if err != nil {
		fmt.Println("✗")
		return err
	}
	// Show truncated key for confirmation
	keyParts := strings.Fields(sshPubKey)
	keyType := keyParts[0]
	keyPreview := keyParts[1]
	if len(keyPreview) > 16 {
		keyPreview = keyPreview[:8] + "..." + keyPreview[len(keyPreview)-8:]
	}
	fmt.Printf("%s %s\n", keyType, keyPreview)

	// Step 3: Initiate shell auth
	fmt.Println("\nInitiating authentication...")
	initResp, err := auth.ShellAuthInitiate(email, sshPubKey)
	if err != nil {
		wrapped := fmt.Errorf("failed to initiate auth: %w", err)
		trackSemanticOutcome("shell_auth_email_sent", wrapped, map[string]any{
			"flow":   "login",
			"method": "shell_auth",
		})
		return wrapped
	}

	var verifyResp *auth.ShellAuthVerifyResponse

	if initResp.AlreadyVerified {
		// Key is already verified — skip email, go straight to verify-key for org list
		fmt.Println("SSH key already verified ✓")
		verifyResp, err = auth.ShellAuthCheckVerification(email, sshPubKey)
		if err != nil {
			wrapped := fmt.Errorf("failed to fetch org list: %w", err)
			trackSemanticOutcome("shell_auth_email_verified", wrapped, map[string]any{
				"flow":             "login",
				"method":           "shell_auth",
				"already_verified": true,
			})
			return wrapped
		}
		trackSemanticEvent("shell_auth_email_verified", map[string]any{
			"flow":             "login",
			"method":           "shell_auth",
			"already_verified": true,
		})
	} else {
		if initResp.IsNewUser {
			fmt.Println("Creating new Vers account...")
		}

		// Step 4: Wait for email verification
		fmt.Printf("\n📧 Verification email sent to %s\n", email)
		fmt.Println("   Click the link in the email to continue.")
		fmt.Print("   Waiting for verification...")
		trackSemanticEvent("shell_auth_email_sent", map[string]any{
			"flow":        "login",
			"method":      "shell_auth",
			"is_new_user": initResp.IsNewUser,
		})

		verifyResp, err = auth.ShellAuthPollVerification(email, sshPubKey, 10*time.Minute)
		if err != nil {
			fmt.Println(" ✗")
			trackSemanticOutcome("shell_auth_email_verified", err, map[string]any{
				"flow":        "login",
				"method":      "shell_auth",
				"is_new_user": initResp.IsNewUser,
			})
			return err
		}
		fmt.Println(" ✓")
		trackSemanticEvent("shell_auth_email_verified", map[string]any{
			"flow":        "login",
			"method":      "shell_auth",
			"is_new_user": initResp.IsNewUser,
		})
	}

	// Step 5: Select organization
	orgName := ""
	orgSelectionMode := "interactive"
	if len(verifyResp.Orgs) == 0 {
		err := fmt.Errorf("no organizations found for this account")
		trackSemanticOutcome("shell_auth_org_selected", err, map[string]any{
			"flow":      "login",
			"method":    "shell_auth",
			"org_count": len(verifyResp.Orgs),
		})
		return err
	} else if len(verifyResp.Orgs) == 1 {
		orgName = verifyResp.Orgs[0].Name
		orgSelectionMode = "single_option"
		fmt.Printf("\nOrganization: %s\n", orgName)
	} else {
		fmt.Println("\nSelect an organization:")
		for i, org := range verifyResp.Orgs {
			fmt.Printf("  [%d] %s (%s)\n", i+1, org.Name, org.Role)
		}
		fmt.Print("Enter number: ")
		reader := bufio.NewReader(os.Stdin)
		input, err := reader.ReadString('\n')
		if err != nil {
			wrapped := fmt.Errorf("failed to read input: %w", err)
			trackSemanticOutcome("shell_auth_org_selected", wrapped, map[string]any{
				"flow":      "login",
				"method":    "shell_auth",
				"org_count": len(verifyResp.Orgs),
			})
			return wrapped
		}
		input = strings.TrimSpace(input)
		var choice int
		if _, err := fmt.Sscanf(input, "%d", &choice); err != nil || choice < 1 || choice > len(verifyResp.Orgs) {
			wrapped := fmt.Errorf("invalid selection")
			trackSemanticOutcome("shell_auth_org_selected", wrapped, map[string]any{
				"flow":      "login",
				"method":    "shell_auth",
				"org_count": len(verifyResp.Orgs),
			})
			return wrapped
		}
		orgName = verifyResp.Orgs[choice-1].Name
	}
	trackSemanticEvent("shell_auth_org_selected", map[string]any{
		"flow":               "login",
		"method":             "shell_auth",
		"org_selection_mode": orgSelectionMode,
		"org_count":          len(verifyResp.Orgs),
	})

	// Step 6: Create API key
	hostname, _ := os.Hostname()
	label := fmt.Sprintf("vers-cli-%s", hostname)
	// Ensure label is at least 5 characters
	if len(label) < 5 {
		label = "vers-cli-key"
	}

	fmt.Print("Creating API key... ")
	keyResp, err := auth.ShellAuthCreateAPIKey(email, sshPubKey, label, orgName)
	if err != nil {
		fmt.Println("✗")
		wrapped := fmt.Errorf("failed to create API key: %w", err)
		trackSemanticOutcome("api_key_validated", wrapped, map[string]any{
			"flow":   "login",
			"method": "shell_auth",
			"stage":  "create",
		})
		return wrapped
	}
	fmt.Println("✓")

	// Step 7: Validate and save
	fmt.Print("Validating API key... ")
	if err := validateAPIKey(keyResp.APIKey); err != nil {
		fmt.Println("✗")
		trackSemanticOutcome("api_key_validated", err, map[string]any{
			"flow":   "login",
			"method": "shell_auth",
			"stage":  "validate",
		})
		return err
	}
	trackSemanticEvent("api_key_validated", map[string]any{
		"flow":   "login",
		"method": "shell_auth",
		"stage":  "validate",
	})
	config, err := auth.LoadConfig()
	if err != nil {
		return fmt.Errorf("error loading config: %w", err)
	}
	config.APIKey = keyResp.APIKey
	config.Email = email
	config.UserID = verifyResp.UserID
	config.OrgID = keyResp.OrgID
	if telemetryClient != nil && telemetryClient.AnonymousID() != "" {
		config.AnonymousID = telemetryClient.AnonymousID()
	}
	if err := auth.SaveConfig(config); err != nil {
		return fmt.Errorf("error saving config: %w", err)
	}
	if telemetryClient != nil {
		telemetryClient.ReplaceConfig(config)
		telemetryClient.SetAPIKey(keyResp.APIKey)
		telemetryClient.TrackAuthSuccess("signin_completed", "shell_auth", verifyResp.UserID, keyResp.OrgID, email)
	}

	fmt.Printf("\n✓ Successfully authenticated with Vers (org: %s)\n", keyResp.OrgName)
	return nil
}

// loginCmd represents the login command
var loginCmd = &cobra.Command{
	Use:   "login",
	Short: "Authenticate with the Vers platform",
	Long: `Login to the Vers platform.

There are three ways to authenticate:

  vers login --git      Use your git email and SSH key (recommended)
  vers login --token    Provide an existing API key
  vers login            Prompt for an API key

The --git flag uses Shell Auth to create an API key automatically.
It reads your email from git config and finds your SSH public key,
then sends a verification email. Click the link and you're in.`,
	RunE: func(cmd *cobra.Command, args []string) (retErr error) {
		if loginGit {
			return loginWithGit()
		}

		if telemetryClient != nil {
			telemetryClient.BeginAuthenticationFlow()
		}
		defer func() {
			if retErr != nil {
				trackAuthFailure("signin_completed", "login", "api_key", retErr)
			}
		}()

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
			trackSemanticOutcome("api_key_validated", err, map[string]any{
				"flow":   "login",
				"method": "api_key",
			})
			return err // Stop here if validation fails
		}
		trackSemanticEvent("api_key_validated", map[string]any{
			"flow":   "login",
			"method": "api_key",
		})

		// Save the API key only if validation succeeded. Clear any stale user-scoped
		// cache so later captures rely on server-side canonical attribution until a
		// trusted identity backfill happens.
		config, err := auth.LoadConfig()
		if err != nil {
			return fmt.Errorf("error loading config: %w", err)
		}
		config.APIKey = token
		auth.ClearUserIdentity(config)
		if telemetryClient != nil && telemetryClient.AnonymousID() != "" {
			config.AnonymousID = telemetryClient.AnonymousID()
		}
		if err := auth.SaveConfig(config); err != nil {
			return fmt.Errorf("error saving API key: %w", err)
		}
		if telemetryClient != nil {
			telemetryClient.ReplaceConfig(config)
			telemetryClient.SetAPIKey(token)
			telemetryClient.TrackAuthSuccess("signin_completed", "api_key", "", "", "")
		}

		fmt.Println("Successfully authenticated with Vers")
		return nil
	},
}

func init() {
	rootCmd.AddCommand(loginCmd)
	loginCmd.Flags().StringVarP(&token, "token", "t", "", "API token for authentication")
	loginCmd.Flags().BoolVar(&loginGit, "git", false, "Authenticate using your git email and SSH key")
}
