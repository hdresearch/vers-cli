package cmd

import (
	"time"

	"github.com/hdresearch/vers-cli/internal/handlers"
	"github.com/spf13/cobra"
)

var upgradeCmd = &cobra.Command{
	Use:   "upgrade",
	Short: "Upgrade vers CLI to the latest version",
	Long: `Check for and install the latest version of the vers CLI from GitHub releases.

This command will:
- Check the current version against the latest GitHub release
- Download and install the new version if available
- Verify the download using SHA256 checksums
- Preserve the current binary as a backup

Examples:
  vers upgrade                    # Check and upgrade to latest version
  vers upgrade --check-only       # Only check for updates, don't install
  vers upgrade --prerelease       # Include pre-release versions
	vers upgrade --skip-checksum    # Skip checksum verification (not recommended)`,
	RunE: func(cmd *cobra.Command, args []string) error {
		started := time.Now()
		req := handlers.UpgradeReq{
			CurrentVersion: Version,
			Repository:     Repository,
			CheckOnly:      checkOnly,
			Prerelease:     prerelease,
			SkipChecksum:   skipChecksum,
			OnUpgradeOutcome: func(fromVersion, toVersion string, err error) {
				trackSemanticOutcome("cli_updated", err, map[string]any{
					"from_version":  fromVersion,
					"to_version":    toVersion,
					"skip_checksum": skipChecksum,
				})
			},
		}
		err := handlers.HandleUpgrade(application, req)
		trackSemanticOutcomeWithDuration("cli_update_checked", err, started, map[string]any{
			"check_only":    checkOnly,
			"prerelease":    prerelease,
			"skip_checksum": skipChecksum,
		})
		return err
	},
}

var (
	checkOnly    bool
	prerelease   bool
	skipChecksum bool
)

func init() {
	rootCmd.AddCommand(upgradeCmd)
	upgradeCmd.Flags().BoolVar(&checkOnly, "check-only", false, "Only check for updates without installing")
	upgradeCmd.Flags().BoolVar(&prerelease, "prerelease", false, "Include pre-release versions")
	upgradeCmd.Flags().BoolVar(&skipChecksum, "skip-checksum", false, "Skip SHA256 checksum verification (not recommended)")
}
