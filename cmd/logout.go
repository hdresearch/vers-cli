package cmd

import (
	"fmt"

	"github.com/hdresearch/vers-cli/internal/auth"
	"github.com/spf13/cobra"
)

// logoutCmd represents the logout command
var logoutCmd = &cobra.Command{
	Use:   "logout",
	Short: "Log out from the Vers platform",
	Long:  `Log out from the Vers platform by removing your stored API key.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		// Check if an API key is present
		hasKey, err := auth.HasAPIKey()
		if err != nil {
			wrapped := fmt.Errorf("error checking for API key: %w", err)
			trackSemanticOutcome("signout", wrapped, nil)
			return wrapped
		}

		if !hasKey {
			fmt.Println("You are not currently logged in")
			trackSemanticEvent("signout", map[string]any{
				"had_active_session": false,
			})
			return nil
		}

		// Clear API key AND user-scoped identity so a subsequent login as a
		// different user on the same machine doesn't get misattributed.
		// Keep AnonymousID / DeviceID — those are device-level, not user-level.
		config, err := auth.LoadConfig()
		if err != nil {
			wrapped := fmt.Errorf("error loading config: %w", err)
			trackSemanticOutcome("signout", wrapped, map[string]any{
				"had_active_session": true,
			})
			return wrapped
		}
		config.APIKey = ""
		auth.ClearUserIdentity(config)
		config.AnonymousID = auth.NewTelemetryID()
		if err := auth.SaveConfig(config); err != nil {
			wrapped := fmt.Errorf("error removing credentials: %w", err)
			trackSemanticOutcome("signout", wrapped, map[string]any{
				"had_active_session": true,
			})
			return wrapped
		}
		trackSemanticEvent("signout", map[string]any{
			"had_active_session": true,
		})
		if telemetryClient != nil {
			telemetryClient.ReplaceConfig(config)
			telemetryClient.SetAPIKey("")
		}

		fmt.Println("Successfully logged out from Vers")
		return nil
	},
}

func init() {
	rootCmd.AddCommand(logoutCmd)
}
