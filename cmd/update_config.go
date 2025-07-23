package cmd

import (
	"fmt"
	"time"

	"github.com/hdresearch/vers-cli/internal/config"
	"github.com/spf13/cobra"
)

var updateConfigCmd = &cobra.Command{
	Use:   "update-config",
	Short: "Manage update checking configuration",
	Long: `Configure how vers checks for updates.

This command allows you to:
- View current update checking settings
- Change the update check interval
- Clear skipped versions
- Reset update checking state

Examples:
  vers update-config                           # Show current settings
  vers update-config --interval 7200          # Check every 2 hours
  vers update-config --interval 86400         # Check daily
  vers update-config --clear-skipped          # Clear skipped versions
  vers update-config --reset                  # Reset all update state`,
	RunE: runUpdateConfig,
}

var (
	showConfig   bool
	setInterval  int64
	clearSkipped bool
	resetState   bool
)

func init() {
	rootCmd.AddCommand(updateConfigCmd)

	updateConfigCmd.Flags().Int64Var(&setInterval, "interval", 0, "Set update check interval in seconds (0 to show current)")
	updateConfigCmd.Flags().BoolVar(&clearSkipped, "clear-skipped", false, "Clear skipped version")
	updateConfigCmd.Flags().BoolVar(&resetState, "reset", false, "Reset all update checking state")
}

func runUpdateConfig(cmd *cobra.Command, args []string) error {
	cliConfig, err := config.LoadCLIConfig()
	if err != nil {
		return fmt.Errorf("failed to load CLI config: %w", err)
	}

	// Handle reset flag
	if resetState {
		cliConfig.UpdateCheck = config.DefaultUpdateCheckConfig()
		if err := config.SaveCLIConfig(cliConfig); err != nil {
			return fmt.Errorf("failed to save config: %w", err)
		}
		fmt.Println("✓ Update checking state has been reset to defaults")
		return nil
	}

	// Handle clear skipped flag
	if clearSkipped {
		cliConfig.UpdateCheck.SkippedVersion = ""
		if err := config.SaveCLIConfig(cliConfig); err != nil {
			return fmt.Errorf("failed to save config: %w", err)
		}
		fmt.Println("✓ Cleared skipped version")
	}

	// Handle interval setting
	if setInterval > 0 {
		cliConfig.UpdateCheck.CheckInterval = setInterval
		// Recalculate next check time with new interval
		cliConfig.SetNextCheckTime()
		if err := config.SaveCLIConfig(cliConfig); err != nil {
			return fmt.Errorf("failed to save config: %w", err)
		}
		fmt.Printf("✓ Update check interval set to %s\n", formatDuration(time.Duration(setInterval)*time.Second))
	}

	// Show current configuration
	fmt.Println("Update Configuration:")
	fmt.Printf("  Check Interval: %s\n", formatDuration(time.Duration(cliConfig.UpdateCheck.CheckInterval)*time.Second))
	fmt.Printf("  Last Check: %s\n", formatTime(cliConfig.UpdateCheck.LastCheck))
	fmt.Printf("  Next Check: %s\n", formatTime(cliConfig.UpdateCheck.NextCheck))

	if cliConfig.UpdateCheck.AvailableVersion != "" {
		fmt.Printf("  Available Version: %s\n", cliConfig.UpdateCheck.AvailableVersion)
	} else {
		fmt.Printf("  Available Version: none\n")
	}

	if cliConfig.UpdateCheck.SkippedVersion != "" {
		fmt.Printf("  Skipped Version: %s\n", cliConfig.UpdateCheck.SkippedVersion)
	} else {
		fmt.Printf("  Skipped Version: none\n")
	}

	return nil
}

func formatDuration(d time.Duration) string {
	if d == 0 {
		return "not set"
	}

	hours := int(d.Hours())
	minutes := int(d.Minutes()) % 60

	if hours > 0 {
		if minutes > 0 {
			return fmt.Sprintf("%dh %dm", hours, minutes)
		}
		return fmt.Sprintf("%dh", hours)
	}

	if minutes > 0 {
		return fmt.Sprintf("%dm", minutes)
	}

	return fmt.Sprintf("%.0fs", d.Seconds())
}

func formatTime(t time.Time) string {
	if t.IsZero() {
		return "never"
	}

	if time.Since(t) > 0 {
		// Past time
		return fmt.Sprintf("%s (%s ago)", t.Format("2006-01-02 15:04:05"), time.Since(t).Round(time.Minute))
	} else {
		// Future time
		return fmt.Sprintf("%s (in %s)", t.Format("2006-01-02 15:04:05"), time.Until(t).Round(time.Minute))
	}
}
