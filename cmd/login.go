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
func loginWithGit() error {
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
		return fmt.Errorf("failed to initiate auth: %w", err)
	}

	var verifyResp *auth.ShellAuthVerifyResponse

	if initResp.AlreadyVerified {
		// Key is already verified — skip email, go straight to verify-key for org list
		fmt.Println("SSH key already verified ✓")
		verifyResp, err = auth.ShellAuthCheckVerification(email, sshPubKey)
		if err != nil {
			return fmt.Errorf("failed to fetch org list: %w", err)
		}
	} else {
		if initResp.IsNewUser {
			fmt.Println("Creating new Vers account...")
		}

		// Step 4: Wait for email verification
		fmt.Printf("\n📧 Verification email sent to %s\n", email)
		fmt.Println("   Click the link in the email to continue.")
		fmt.Print("   Waiting for verification...")

		verifyResp, err = auth.ShellAuthPollVerification(email, sshPubKey, 10*time.Minute)
		if err != nil {
			fmt.Println(" ✗")
			return err
		}
		fmt.Println(" ✓")
	}

	// Step 5: Select organization
	orgName := ""
	if len(verifyResp.Orgs) == 0 {
		return fmt.Errorf("no organizations found for this account")
	} else if len(verifyResp.Orgs) == 1 {
		orgName = verifyResp.Orgs[0].Name
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
			return fmt.Errorf("failed to read input: %w", err)
		}
		input = strings.TrimSpace(input)
		var choice int
		if _, err := fmt.Sscanf(input, "%d", &choice); err != nil || choice < 1 || choice > len(verifyResp.Orgs) {
			return fmt.Errorf("invalid selection")
		}
		orgName = verifyResp.Orgs[choice-1].Name
	}

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
		return fmt.Errorf("failed to create API key: %w", err)
	}
	fmt.Println("✓")

	// Step 7: Validate and save
	fmt.Print("Validating API key... ")
	if err := validateAPIKey(keyResp.APIKey); err != nil {
		fmt.Println("✗")
		return err
	}

	if err := auth.SaveAPIKey(keyResp.APIKey); err != nil {
		return fmt.Errorf("error saving API key: %w", err)
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
	RunE: func(cmd *cobra.Command, args []string) error {
		if loginGit {
			return loginWithGit()
		}

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
	loginCmd.Flags().BoolVar(&loginGit, "git", false, "Authenticate using your git email and SSH key")
}
