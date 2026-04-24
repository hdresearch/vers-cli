package cmd

import (
	"fmt"

	"github.com/hdresearch/vers-cli/internal/auth"
	"github.com/spf13/cobra"
)

var telemetryCmd = &cobra.Command{
	Use:   "telemetry",
	Short: "Manage CLI telemetry settings",
	Long: `Manage CLI telemetry settings.

Telemetry is enabled by default. You can disable it locally in .versrc,
or override it for a process with DO_NOT_TRACK=1.`,
}

var telemetryEnableCmd = &cobra.Command{
	Use:   "enable",
	Short: "Enable CLI telemetry in .versrc",
	RunE: func(cmd *cobra.Command, args []string) error {
		config, err := auth.LoadConfig()
		if err != nil {
			if telemetryClient != nil {
				telemetryClient.TrackToggle(true, false, err)
			}
			return err
		}
		telemetryConfig := auth.EnsureTelemetryConfig(config)
		enabled := true
		telemetryConfig.Enabled = &enabled
		telemetryConfig.NoticeAcknowledged = true
		if err := auth.SaveConfig(config); err != nil {
			if telemetryClient != nil {
				telemetryClient.TrackToggle(true, false, err)
			}
			return err
		}
		if telemetryClient != nil {
			telemetryClient.TrackToggle(true, true, nil)
		}
		fmt.Println("Telemetry enabled.")
		return nil
	},
}

var telemetryDisableCmd = &cobra.Command{
	Use:   "disable",
	Short: "Disable CLI telemetry in .versrc",
	RunE: func(cmd *cobra.Command, args []string) error {
		config, err := auth.LoadConfig()
		if err != nil {
			if telemetryClient != nil {
				telemetryClient.TrackToggle(false, false, err)
			}
			return err
		}
		telemetryConfig := auth.EnsureTelemetryConfig(config)
		enabled := false
		telemetryConfig.Enabled = &enabled
		telemetryConfig.NoticeAcknowledged = true
		if err := auth.SaveConfig(config); err != nil {
			if telemetryClient != nil {
				telemetryClient.TrackToggle(false, false, err)
			}
			return err
		}
		// Track after the config write succeeds. TrackToggle bypasses the
		// enabled gate for this single event, so this still emits the last
		// event before going dark unless DO_NOT_TRACK is set.
		if telemetryClient != nil {
			telemetryClient.TrackToggle(false, true, nil)
		}
		fmt.Println("Telemetry disabled.")
		return nil
	},
}

var telemetryStatusCmd = &cobra.Command{
	Use:   "status",
	Short: "Show effective CLI telemetry status",
	RunE: func(cmd *cobra.Command, args []string) error {
		config, err := auth.LoadConfig()
		if err != nil {
			trackSemanticOutcome("telemetry_status_viewed", err, nil)
			return err
		}
		enabled, reason := auth.EffectiveTelemetryStatus(config)
		configured := "default"
		if config.Telemetry != nil && config.Telemetry.Enabled != nil {
			if *config.Telemetry.Enabled {
				configured = "enabled"
			} else {
				configured = "disabled"
			}
		}

		state := "disabled"
		if enabled {
			state = "enabled"
		}
		fmt.Printf("Telemetry: %s\n", state)
		fmt.Printf("Reason: %s\n", reason)
		fmt.Printf("Config: %s\n", configured)
		if noticeAcknowledged(config) {
			fmt.Println("Notice: acknowledged")
		} else {
			fmt.Println("Notice: not yet acknowledged")
		}
		if reason == "disabled by DO_NOT_TRACK" {
			fmt.Println("Note: environment variables currently override .versrc.")
		}
		trackSemanticEvent("telemetry_status_viewed", map[string]any{
			"enabled": enabled,
		})
		return nil
	},
}

func noticeAcknowledged(config *auth.Config) bool {
	return config != nil && config.Telemetry != nil && config.Telemetry.NoticeAcknowledged
}

func init() {
	telemetryCmd.AddCommand(telemetryEnableCmd)
	telemetryCmd.AddCommand(telemetryDisableCmd)
	telemetryCmd.AddCommand(telemetryStatusCmd)
	rootCmd.AddCommand(telemetryCmd)
}
