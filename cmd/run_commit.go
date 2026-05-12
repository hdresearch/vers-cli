package cmd

import (
	"context"
	"fmt"
	"os"
	"strings"

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
	runCommitIsRef  bool
)

// runCommitCmd represents the run-commit command
var runCommitCmd = &cobra.Command{
	Use:   "run-commit [commit-key | repo:tag]",
	Short: "Start a development environment from a commit",
	Long: `Start a Vers development environment from an existing commit.

The argument is treated as a commit ID (UUID) by default. Pass --ref to interpret
it as a repository reference in "repo_name:tag_name" format instead — the API
will resolve the tag to the commit it currently points at within your own org.

Examples:
  vers run-commit c123456789abcdef
  vers run-commit my-app:latest --ref
  vers run-commit my-app:latest --ref --vm-alias dev --wait

Use --json for machine-readable output.
Use --wait to block until the VM is running.`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		commitKey := args[0]

		// Friendly nudge for the common gotcha: user passed something that
		// looks like a repo:tag ref without --ref.
		if !runCommitIsRef && looksLikeRepoRef(commitKey) {
			return fmt.Errorf(
				"'%s' looks like a repo:tag reference; did you mean '--ref'?\n"+
					"  vers run-commit %s --ref",
				commitKey, commitKey,
			)
		}

		cfg, err := runconfig.Load()
		if err != nil {
			return err
		}
		applyFlagOverrides(cmd, cfg)
		apiCtx, cancel := context.WithTimeout(context.Background(), application.Timeouts.APILong)
		defer cancel()
		req := handlers.RunCommitReq{
			CommitKey: commitKey,
			VMAlias:   commitVmAlias,
			Wait:      runCommitWait,
			IsRef:     runCommitIsRef,
		}
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

// looksLikeRepoRef returns true for strings in "word:word" shape where the
// parts contain only characters that are legal in repo/tag names. Used only
// to guess user intent and suggest --ref; never to actually dispatch.
func looksLikeRepoRef(s string) bool {
	i := strings.IndexByte(s, ':')
	if i <= 0 || i == len(s)-1 {
		return false
	}
	legal := func(c byte) bool {
		return (c >= 'a' && c <= 'z') ||
			(c >= 'A' && c <= 'Z') ||
			(c >= '0' && c <= '9') ||
			c == '-' || c == '_' || c == '.'
	}
	for j := 0; j < len(s); j++ {
		if j == i {
			continue
		}
		if !legal(s[j]) {
			return false
		}
	}
	return true
}

func init() {
	rootCmd.AddCommand(runCommitCmd)

	runCommitCmd.Flags().StringVarP(&commitVmAlias, "vm-alias", "N", "", "Set an alias for the root VM")
	runCommitCmd.Flags().BoolVar(&runCommitJSON, "json", false, "Output as JSON")
	runCommitCmd.Flags().StringVar(&runCommitFormat, "format", "", "Output format (json) [deprecated: use --json]")
	_ = runCommitCmd.Flags().MarkDeprecated("format", "use --json instead")
	runCommitCmd.Flags().BoolVar(&runCommitWait, "wait", false, "Wait until VM is running")
	runCommitCmd.Flags().BoolVar(&runCommitIsRef, "ref", false, "Interpret the argument as a repo:tag reference instead of a commit ID")
}
