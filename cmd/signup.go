package cmd

import (
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/hdresearch/vers-cli/internal/auth"
	"github.com/spf13/cobra"
)

var (
	signupGit   bool
	signupOrg   string
	signupEmail string
)

// signupWithGit authenticates using the Shell Auth flow with git email + SSH key.
func signupWithGit() error {
	// Step 1: Get email
	var email string
	if signupEmail != "" {
		email = signupEmail
		fmt.Printf("Using email: %s\n", email)
	} else {
		fmt.Print("Looking up git email... ")
		var err error
		email, err = auth.GetGitEmail()
		if err != nil {
			fmt.Println("✗")
			return err
		}
		fmt.Println(email)
	}

	// Step 2: Find SSH public key
	fmt.Print("Looking up SSH public key... ")
	sshPubKey, sshKeyPath, err := auth.FindSSHPublicKey()
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
	} else if signupOrg != "" {
		// --org flag provided, match by name
		found := false
		for _, org := range verifyResp.Orgs {
			if strings.EqualFold(org.Name, signupOrg) {
				orgName = org.Name
				found = true
				break
			}
		}
		if !found {
			names := make([]string, len(verifyResp.Orgs))
			for i, org := range verifyResp.Orgs {
				names[i] = org.Name
			}
			return fmt.Errorf("organization %q not found. Available: %s", signupOrg, strings.Join(names, ", "))
		}
		fmt.Printf("\nOrganization: %s\n", orgName)
	} else if len(verifyResp.Orgs) == 1 {
		orgName = verifyResp.Orgs[0].Name
		fmt.Printf("\nOrganization: %s\n", orgName)
	} else {
		names := make([]string, len(verifyResp.Orgs))
		for i, org := range verifyResp.Orgs {
			names[i] = org.Name
		}
		return fmt.Errorf("multiple organizations found. Use --org to specify one: %s", strings.Join(names, ", "))
	}

	// Step 6: Create API key
	hostname, _ := os.Hostname()
	label := fmt.Sprintf("vers-cli-%s", hostname)
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

	config, err := auth.LoadConfig()
	if err != nil {
		return fmt.Errorf("error loading config: %w", err)
	}
	config.APIKey = keyResp.APIKey
	config.Email = email
	config.SSHKeyPath = sshKeyPath
	if err := auth.SaveConfig(config); err != nil {
		return fmt.Errorf("error saving config: %w", err)
	}

	fmt.Printf("\n✓ Successfully authenticated with Vers (org: %s)\n", keyResp.OrgName)
	return nil
}

var signupCmd = &cobra.Command{
	Use:   "signup",
	Short: "Create a Vers account and authenticate",
	Long: `Sign up for the Vers platform using your email and SSH key.

By default, signup uses your git email and SSH public key to create
an account. A verification email is sent — click the link and you're in.

  vers signup                        Sign up with git email + SSH key (default)
  vers signup --email you@example.com  Use a specific email instead of git config
  vers signup --org myorg            Pick org non-interactively (for scripts/agents)
  vers signup --git=false            Prompt for an API key instead

If you already have an account, this will log you in.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		if signupGit {
			return signupWithGit()
		}

		// Fallback: prompt for API key (same as `vers login`)
		apiKey, err := secureReadAPIKey()
		if err != nil {
			return err
		}

		fmt.Print("Validating API key... ")
		if err := validateAPIKey(apiKey); err != nil {
			fmt.Println("✗")
			return err
		}
		fmt.Println("✓")

		if err := auth.SaveAPIKey(apiKey); err != nil {
			return fmt.Errorf("error saving API key: %w", err)
		}

		fmt.Println("\n✓ Successfully authenticated with Vers")
		return nil
	},
}

func init() {
	rootCmd.AddCommand(signupCmd)
	signupCmd.Flags().BoolVar(&signupGit, "git", true, "Authenticate using your git email and SSH key (default: true)")
	signupCmd.Flags().StringVar(&signupOrg, "org", "", "Organization name (skips interactive selection)")
	signupCmd.Flags().StringVar(&signupEmail, "email", "", "Email address (overrides git config user.email)")
}
