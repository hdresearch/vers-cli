package cmd

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"sort"
	"strings"
	"text/tabwriter"

	"github.com/hdresearch/vers-cli/internal/handlers"
	pres "github.com/hdresearch/vers-cli/internal/presenters"
	"github.com/spf13/cobra"
	"golang.org/x/term"
)

var secretFormat string

var secretCmd = &cobra.Command{
	Use:   "secret",
	Short: "Manage secrets",
	Long: `Manage secrets that are securely injected into VMs at startup.

Secrets are written to /etc/environment in newly created VMs, where they
are available to all processes (SSH sessions, exec'd commands, dev servers).

On cross-account operations (restoring or branching from another user's
public commit), secrets from the original owner are automatically cleared
and replaced with yours.

Use subcommands to list, set, or delete secrets.

Examples:
  vers secret set ANTHROPIC_API_KEY
  vers secret set DATABASE_URL postgres://localhost/mydb
  vers secret list
  vers secret delete OLD_TOKEN`,
	Aliases: []string{"secrets"},
}

var secretListCmd = &cobra.Command{
	Use:   "list",
	Short: "List all secrets",
	Long: `List all secrets configured for your account.

Secret values are masked by default. Use --reveal to show full values.`,
	Aliases: []string{"ls"},
	RunE: func(cmd *cobra.Command, args []string) error {
		reveal, _ := cmd.Flags().GetBool("reveal")

		apiCtx, cancel := context.WithTimeout(context.Background(), application.Timeouts.APIMedium)
		defer cancel()

		vars, err := handlers.HandleEnvList(apiCtx, application, handlers.EnvListReq{})
		if err != nil {
			return err
		}

		format := pres.ParseFormat(false, secretFormat)
		switch format {
		case pres.FormatJSON:
			if reveal {
				pres.PrintJSON(vars)
			} else {
				masked := make(map[string]string, len(vars))
				for k, v := range vars {
					masked[k] = maskValue(v)
				}
				pres.PrintJSON(masked)
			}
		default:
			if len(vars) == 0 {
				fmt.Println("No secrets configured.")
				fmt.Println("")
				fmt.Println("Set one with: vers secret set MY_API_KEY")
				return nil
			}

			keys := make([]string, 0, len(vars))
			for k := range vars {
				keys = append(keys, k)
			}
			sort.Strings(keys)

			w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
			fmt.Fprintln(w, "KEY\tVALUE")
			for _, key := range keys {
				value := vars[key]
				if !reveal {
					value = maskValue(value)
				}
				fmt.Fprintf(w, "%s\t%s\n", key, value)
			}
			w.Flush()
		}
		return nil
	},
}

var secretSetCmd = &cobra.Command{
	Use:   "set KEY [VALUE]",
	Short: "Set a secret",
	Long: `Set a secret that will be injected into newly created VMs.

If VALUE is omitted, you'll be prompted to enter it (hidden input).
If stdin is piped, the value is read from stdin.

The key must be a valid shell identifier (letters, digits, underscores only,
cannot start with a digit).

Examples:
  vers secret set ANTHROPIC_API_KEY              # prompts for value
  vers secret set DATABASE_URL postgres://...     # inline value
  echo "sk-ant-..." | vers secret set API_KEY     # from pipe
  cat .env.secret | vers secret set API_KEY       # from file`,
	Args: cobra.RangeArgs(1, 2),
	RunE: func(cmd *cobra.Command, args []string) error {
		key := args[0]

		if key == "" {
			return fmt.Errorf("key cannot be empty")
		}
		if !isValidEnvKey(key) {
			return fmt.Errorf("invalid key '%s': must start with letter or underscore, contain only letters, digits, and underscores", key)
		}

		var value string

		if len(args) == 2 {
			// Value provided as argument
			value = args[1]
		} else {
			// Read value from stdin (piped) or prompt (interactive)
			var err error
			value, err = readSecretValue(key)
			if err != nil {
				return fmt.Errorf("failed to read secret value: %w", err)
			}
		}

		if value == "" {
			return fmt.Errorf("secret value cannot be empty")
		}

		apiCtx, cancel := context.WithTimeout(context.Background(), application.Timeouts.APIMedium)
		defer cancel()

		err := handlers.HandleEnvSet(apiCtx, application, handlers.EnvSetReq{
			Key:   key,
			Value: value,
		})
		if err != nil {
			return err
		}

		fmt.Printf("Secret %s set successfully.\n", key)
		fmt.Println("This secret will be available in newly created VMs.")
		return nil
	},
}

var secretDeleteCmd = &cobra.Command{
	Use:   "delete KEY",
	Short: "Delete a secret",
	Long: `Delete a secret.

This removes the secret from your configuration. It will no longer be
injected into newly created VMs (existing VMs are not affected).

Examples:
  vers secret delete OLD_API_KEY
  vers secret rm DATABASE_URL`,
	Aliases: []string{"del", "rm", "remove"},
	Args:    cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		key := args[0]

		if key == "" {
			return fmt.Errorf("key cannot be empty")
		}

		apiCtx, cancel := context.WithTimeout(context.Background(), application.Timeouts.APIMedium)
		defer cancel()

		err := handlers.HandleEnvDelete(apiCtx, application, handlers.EnvDeleteReq{
			Key: key,
		})
		if err != nil {
			return err
		}

		fmt.Printf("Secret %s deleted.\n", key)
		return nil
	},
}

// readSecretValue reads a secret value from stdin.
// If stdin is a terminal, prompts with hidden input.
// If stdin is piped, reads the first line.
func readSecretValue(key string) (string, error) {
	if term.IsTerminal(int(os.Stdin.Fd())) {
		// Interactive — prompt with hidden input
		fmt.Fprintf(os.Stderr, "Enter value for %s: ", key)
		raw, err := term.ReadPassword(int(os.Stdin.Fd()))
		fmt.Fprintln(os.Stderr) // newline after hidden input
		if err != nil {
			return "", err
		}
		return strings.TrimSpace(string(raw)), nil
	}

	// Piped — read from stdin
	scanner := bufio.NewScanner(os.Stdin)
	if scanner.Scan() {
		return strings.TrimSpace(scanner.Text()), nil
	}
	if err := scanner.Err(); err != nil {
		return "", err
	}
	return "", fmt.Errorf("no input received on stdin")
}

// maskValue masks a secret value for display, showing only a prefix hint.
func maskValue(value string) string {
	if len(value) <= 4 {
		return "****"
	}
	if len(value) <= 8 {
		return value[:2] + "****"
	}
	return value[:4] + "****" + value[len(value)-2:]
}

func init() {
	rootCmd.AddCommand(secretCmd)

	secretCmd.AddCommand(secretListCmd)
	secretCmd.AddCommand(secretSetCmd)
	secretCmd.AddCommand(secretDeleteCmd)

	secretListCmd.Flags().StringVar(&secretFormat, "format", "", "Output format (json)")
	secretListCmd.Flags().Bool("reveal", false, "Show full secret values (unmasked)")
}
