package cmd

import "github.com/spf13/cobra"

// githubCmd is the parent for GitHub-related subcommands.
var githubCmd = &cobra.Command{
	Use:   "github",
	Short: "GitHub App integration",
	Long: `Manage the Vers GitHub App integration.

Install the GitHub App on your organization and mint short-lived
installation access tokens for CI, agents, and scripts.

Examples:
  vers github install --org myorg
  vers github mint-token
  vers github mint-token --repo myorg/my-repo --format json`,
}

func init() {
	rootCmd.AddCommand(githubCmd)
}
