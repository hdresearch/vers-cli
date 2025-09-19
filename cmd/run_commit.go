package cmd

import (
	"context"

	"github.com/hdresearch/vers-cli/internal/handlers"
	pres "github.com/hdresearch/vers-cli/internal/presenters"
	"github.com/spf13/cobra"
)

var (
	commitClusterAlias string
	commitVmAlias      string
)

// runCommitCmd represents the run-commit command
var runCommitCmd = &cobra.Command{
	Use:   "run-commit [commit-key]",
	Short: "Start a development environment from a commit",
	Long:  `Start a Vers development environment from an existing commit using its commit key.`,
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		commitKey := args[0]
		cfg, err := loadConfig()
		if err != nil {
			return err
		}
		applyFlagOverrides(cmd, cfg)
		apiCtx, cancel := context.WithTimeout(context.Background(), application.Timeouts.APIMedium)
		defer cancel()
		req := handlers.RunCommitReq{CommitKey: commitKey, FsSizeClusterMiB: cfg.Machine.FsSizeClusterMib, ClusterAlias: commitClusterAlias, VMAlias: commitVmAlias}
		view, err := handlers.HandleRunCommit(apiCtx, application, req)
		if err != nil {
			return err
		}
		pres.RenderRunCommit(application, view)
		return nil
	},
}

// StartClusterFromCommit starts a development environment from an existing commit
// start-from-commit logic moved to internal/handlers/run_commit.go

func init() {
	rootCmd.AddCommand(runCommitCmd)

	// Add flags to override configuration (same as run command)
	runCommitCmd.Flags().Int64("fs-size-cluster", 0, "Override cluster filesystem size (MiB)")
	runCommitCmd.Flags().StringVarP(&commitClusterAlias, "cluster-alias", "n", "", "Set an alias for the cluster")
	runCommitCmd.Flags().StringVarP(&commitVmAlias, "vm-alias", "N", "", "Set an alias for the root VM")
}
