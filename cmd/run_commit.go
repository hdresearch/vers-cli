package cmd

import (
	"context"
	"os"

	"github.com/hdresearch/vers-cli/internal/handlers"
	"github.com/hdresearch/vers-cli/internal/jobs"
	pres "github.com/hdresearch/vers-cli/internal/presenters"
	"github.com/hdresearch/vers-cli/internal/runconfig"
	"github.com/spf13/cobra"
)

var (
	commitVmAlias   string
	runCommitJSON   bool
	runCommitFormat string
	runCommitWait   bool
)

// runCommitCmd represents the run-commit command
var runCommitCmd = &cobra.Command{
	Use:   "run-commit [commit-key]",
	Short: "Start a development environment from a commit",
	Long: `Start a Vers development environment from an existing commit using its commit key.

Use --json for machine-readable output.
Use --wait to block until the VM is running.`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		commitKey := args[0]
		cfg, err := runconfig.Load()
		if err != nil {
			return err
		}
		applyFlagOverrides(cmd, cfg)
		apiCtx, cancel := context.WithTimeout(context.Background(), application.Timeouts.APILong)
		defer cancel()
		req := handlers.RunCommitReq{CommitKey: commitKey, VMAlias: commitVmAlias, Wait: runCommitWait}
		var jobID string
		if runCommitWait {
			jobID, _ = jobs.Submit(jobs.Submission{
				Kind:    "vm.run_commit",
				Command: "vers run-commit --wait",
				Args:    os.Args[1:],
			})
		}
		view, err := handlers.HandleRunCommit(apiCtx, application, req)
		if err != nil {
			if runCommitWait {
				_ = jobs.Fail(jobID, err)
			}
			return err
		}
		if runCommitWait {
			_ = jobs.Complete(jobID, view.RootVmID)
		}

		format, err := pres.ParseFormat(false, runCommitJSON, runCommitFormat)
		if err != nil {
			return err
		}
		switch format {
		case pres.FormatJSON:
			pres.PrintJSON(view)
		default:
			pres.RenderRunCommit(application, view)
		}
		return nil
	},
}

func init() {
	rootCmd.AddCommand(runCommitCmd)

	runCommitCmd.Flags().StringVarP(&commitVmAlias, "vm-alias", "N", "", "Set an alias for the root VM")
	runCommitCmd.Flags().BoolVar(&runCommitJSON, "json", false, "Output as JSON")
	runCommitCmd.Flags().StringVar(&runCommitFormat, "format", "", "Output format (json) [deprecated: use --json]")
	_ = runCommitCmd.Flags().MarkDeprecated("format", "use --json instead")
	runCommitCmd.Flags().BoolVar(&runCommitWait, "wait", false, "Wait until VM is running")
}
