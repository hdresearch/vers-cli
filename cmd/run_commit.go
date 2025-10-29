package cmd

import (
	"context"

	"github.com/hdresearch/vers-cli/internal/handlers"
	pres "github.com/hdresearch/vers-cli/internal/presenters"
	"github.com/hdresearch/vers-cli/internal/runconfig"
	"github.com/spf13/cobra"
)

var commitVmAlias string

// runCommitCmd represents the run-commit command
var runCommitCmd = &cobra.Command{
	Use:   "run-commit [commit-key]",
	Short: "Start a development environment from a commit",
	Long:  `Start a Vers development environment from an existing commit using its commit key.`,
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		commitKey := args[0]
		cfg, err := runconfig.Load()
		if err != nil {
			return err
		}
		applyFlagOverrides(cmd, cfg)
		apiCtx, cancel := context.WithTimeout(context.Background(), application.Timeouts.APIMedium)
		defer cancel()
		req := handlers.RunCommitReq{CommitKey: commitKey, VMAlias: commitVmAlias}
		view, err := handlers.HandleRunCommit(apiCtx, application, req)
		if err != nil {
			return err
		}
		pres.RenderRunCommit(application, view)
		return nil
	},
}

func init() {
	rootCmd.AddCommand(runCommitCmd)

	runCommitCmd.Flags().StringVarP(&commitVmAlias, "vm-alias", "N", "", "Set an alias for the root VM")
}
