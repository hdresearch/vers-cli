package cmd

import (
	"context"
	"fmt"
	"os"
	"sort"
	"text/tabwriter"

	"github.com/hdresearch/vers-cli/internal/handlers"
	pres "github.com/hdresearch/vers-cli/internal/presenters"
	"github.com/spf13/cobra"
)

var envFormat string

// envCmd represents the env command
var envCmd = &cobra.Command{
	Use:   "env",
	Short: "Manage environment variables",
	Long: `Manage environment variables that are injected into VMs at startup.

Environment variables are written to /etc/environment in newly created VMs,
where they are available for SSH sessions and exec'd processes.

Use subcommands to list, set, or delete environment variables.`,
}

// envListCmd represents the env list command
var envListCmd = &cobra.Command{
	Use:   "list",
	Short: "List all environment variables",
	Long: `List all environment variables configured for your account.

These variables will be injected into newly created VMs at boot time.`,
	Aliases: []string{"ls"},
	RunE: func(cmd *cobra.Command, args []string) error {
		apiCtx, cancel := context.WithTimeout(context.Background(), application.Timeouts.APIMedium)
		defer cancel()

		vars, err := handlers.HandleEnvList(apiCtx, application, handlers.EnvListReq{})
		if err != nil {
			return err
		}

		format := pres.ParseFormat(false, envFormat)
		switch format {
		case pres.FormatJSON:
			pres.PrintJSON(vars)
		default:
			if len(vars) == 0 {
				fmt.Println("No environment variables configured.")
				return nil
			}

			// Sort keys for consistent output
			keys := make([]string, 0, len(vars))
			for k := range vars {
				keys = append(keys, k)
			}
			sort.Strings(keys)

			w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
			fmt.Fprintln(w, "KEY\tVALUE")
			for _, key := range keys {
				value := vars[key]
				// Truncate long values for display
				if len(value) > 50 {
					value = value[:47] + "..."
				}
				fmt.Fprintf(w, "%s\t%s\n", key, value)
			}
			w.Flush()
		}
		return nil
	},
}

// envSetCmd represents the env set command
var envSetCmd = &cobra.Command{
	Use:   "set KEY VALUE",
	Short: "Set an environment variable",
	Long: `Set an environment variable that will be injected into newly created VMs.

The key must be a valid shell identifier (letters, digits, underscores only,
cannot start with a digit). Maximum key length is 256 characters.
Maximum value length is 8192 characters.

Examples:
  vers env set DATABASE_URL postgres://localhost/mydb
  vers env set API_KEY secret123
  vers env set DEBUG true`,
	Args: cobra.ExactArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		key := args[0]
		value := args[1]

		// Basic validation
		if key == "" {
			return fmt.Errorf("key cannot be empty")
		}
		if !isValidEnvKey(key) {
			return fmt.Errorf("invalid key '%s': must start with letter or underscore, contain only letters, digits, and underscores", key)
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

		fmt.Printf("Environment variable %s set successfully.\n", key)
		fmt.Println("This variable will be available in newly created VMs.")
		return nil
	},
}

// envDeleteCmd represents the env delete command
var envDeleteCmd = &cobra.Command{
	Use:   "delete KEY",
	Short: "Delete an environment variable",
	Long: `Delete an environment variable.

This removes the variable from your configuration. It will no longer be
injected into newly created VMs (existing VMs are not affected).

Examples:
  vers env delete DATABASE_URL
  vers env delete API_KEY`,
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

		fmt.Printf("Environment variable %s deleted successfully.\n", key)
		return nil
	},
}

// isValidEnvKey checks if the key is a valid shell identifier
func isValidEnvKey(key string) bool {
	if len(key) == 0 || len(key) > 256 {
		return false
	}

	// Must start with letter or underscore
	first := key[0]
	if !((first >= 'A' && first <= 'Z') || (first >= 'a' && first <= 'z') || first == '_') {
		return false
	}

	// Rest must be letters, digits, or underscores
	for i := 1; i < len(key); i++ {
		c := key[i]
		if !((c >= 'A' && c <= 'Z') || (c >= 'a' && c <= 'z') || (c >= '0' && c <= '9') || c == '_') {
			return false
		}
	}

	return true
}

func init() {
	rootCmd.AddCommand(envCmd)

	// Add subcommands
	envCmd.AddCommand(envListCmd)
	envCmd.AddCommand(envSetCmd)
	envCmd.AddCommand(envDeleteCmd)

	// Add flags
	envListCmd.Flags().StringVar(&envFormat, "format", "", "Output format (json)")
}
