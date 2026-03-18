package cmd

import (
	"context"
	"fmt"

	"github.com/hdresearch/vers-cli/internal/handlers"
	pres "github.com/hdresearch/vers-cli/internal/presenters"
	"github.com/spf13/cobra"
)

var commitFormat string

// commitCmd is the parent command for commit operations.
// When invoked without a subcommand, it commits the current HEAD VM (backward compat).
var commitCmd = &cobra.Command{
	Use:   "commit [vm-id|alias]",
	Short: "Commit the current state of the environment",
	Long: `Save the current state of the Vers environment as a commit.
If no VM ID or alias is provided, commits the current HEAD VM.

Use --format json for machine-readable output.

Subcommands:
  list       List your commits
  delete     Delete a commit
  history    Show the parent commit chain
  publish    Make a commit public
  unpublish  Make a commit private`,
	Args: cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		target := ""
		if len(args) > 0 {
			target = args[0]
		}

		apiCtx, cancel := context.WithTimeout(context.Background(), application.Timeouts.APILong)
		defer cancel()

		res, err := handlers.HandleCommitCreate(apiCtx, application, handlers.CommitCreateReq{
			Target: target,
		})
		if err != nil {
			return err
		}

		format := pres.ParseFormat(false, commitFormat)
		switch format {
		case pres.FormatJSON:
			pres.PrintJSON(res)
		default:
			if res.UsedHEAD {
				fmt.Printf("Using current HEAD VM: %s\n", res.VmID)
			}
			fmt.Printf("✓ Committed VM '%s'\n", res.VmID)
			fmt.Printf("Commit ID: %s\n", res.CommitID)
		}
		return nil
	},
}

var (
	commitListPublic bool
	commitListQuiet  bool
	commitListFormat string
)

var commitListCmd = &cobra.Command{
	Use:   "list",
	Short: "List your commits",
	Long: `List all commits owned by your account. Use --public to list publicly shared commits instead.

Use -q/--quiet to output just commit IDs (one per line), useful for scripting:
  vers commit delete $(vers commit list -q)   # delete all commits

Use --format json for machine-readable output.`,
	Args: cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		apiCtx, cancel := context.WithTimeout(context.Background(), application.Timeouts.APIMedium)
		defer cancel()

		res, err := handlers.HandleCommitsList(apiCtx, application, handlers.CommitsListReq{
			Public: commitListPublic,
		})
		if err != nil {
			return err
		}

		format := pres.ParseFormat(commitListQuiet, commitListFormat)
		switch format {
		case pres.FormatQuiet:
			ids := make([]string, len(res.Commits))
			for i, c := range res.Commits {
				ids[i] = c.CommitID
			}
			pres.PrintQuiet(ids)
		case pres.FormatJSON:
			pres.PrintJSON(res.Commits)
		default:
			pres.RenderCommitsList(application, res)
		}
		return nil
	},
}

var commitDeleteCmd = &cobra.Command{
	Use:   "delete <commit-id>...",
	Short: "Delete one or more commits",
	Long: `Permanently delete one or more commits. Commits must not have any active VMs running from them.

Examples:
  vers commit delete abc-123
  vers commit delete abc-123 def-456
  vers commit delete $(vers commit list -q)   # delete all commits`,
	Args: cobra.MinimumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		apiCtx, cancel := context.WithTimeout(context.Background(), application.Timeouts.APIMedium)
		defer cancel()

		var firstErr error
		for _, id := range args {
			err := handlers.HandleCommitDelete(apiCtx, application, handlers.CommitDeleteReq{
				CommitID: id,
			})
			if err != nil {
				fmt.Fprintf(cmd.ErrOrStderr(), "✗ Failed to delete commit %s: %v\n", id, err)
				if firstErr == nil {
					firstErr = err
				}
				continue
			}
			fmt.Printf("✓ Commit %s deleted\n", id)
		}
		return firstErr
	},
}

var commitHistoryFormat string

var commitHistoryCmd = &cobra.Command{
	Use:   "history <commit-id>",
	Short: "Show the parent commit chain",
	Long: `Display the chain of parent commits leading up to a given commit.

Use --format json for machine-readable output.`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		apiCtx, cancel := context.WithTimeout(context.Background(), application.Timeouts.APIMedium)
		defer cancel()

		res, err := handlers.HandleCommitParents(apiCtx, application, handlers.CommitParentsReq{
			CommitID: args[0],
		})
		if err != nil {
			return err
		}

		format := pres.ParseFormat(false, commitHistoryFormat)
		switch format {
		case pres.FormatJSON:
			pres.PrintJSON(res.Parents)
		default:
			pres.RenderCommitParents(application, res)
		}
		return nil
	},
}

var commitPublishCmd = &cobra.Command{
	Use:   "publish <commit-id>",
	Short: "Make a commit public",
	Long:  `Make a commit publicly accessible so others can restore or branch from it.`,
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		apiCtx, cancel := context.WithTimeout(context.Background(), application.Timeouts.APIMedium)
		defer cancel()

		info, err := handlers.HandleCommitUpdate(apiCtx, application, handlers.CommitUpdateReq{
			CommitID: args[0],
			IsPublic: true,
		})
		if err != nil {
			return err
		}
		fmt.Printf("✓ Commit %s is now public\n", info.CommitID)
		return nil
	},
}

var commitUnpublishCmd = &cobra.Command{
	Use:   "unpublish <commit-id>",
	Short: "Make a commit private",
	Long:  `Make a commit private so only you can access it.`,
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		apiCtx, cancel := context.WithTimeout(context.Background(), application.Timeouts.APIMedium)
		defer cancel()

		info, err := handlers.HandleCommitUpdate(apiCtx, application, handlers.CommitUpdateReq{
			CommitID: args[0],
			IsPublic: false,
		})
		if err != nil {
			return err
		}
		fmt.Printf("✓ Commit %s is now private\n", info.CommitID)
		return nil
	},
}

func init() {
	rootCmd.AddCommand(commitCmd)

	commitCmd.Flags().StringVar(&commitFormat, "format", "", "Output format (json)")

	commitListCmd.Flags().BoolVar(&commitListPublic, "public", false, "List public commits instead of your own")
	commitListCmd.Flags().BoolVarP(&commitListQuiet, "quiet", "q", false, "Only display commit IDs")
	commitListCmd.Flags().StringVar(&commitListFormat, "format", "", "Output format (json)")
	commitCmd.AddCommand(commitListCmd)
	commitCmd.AddCommand(commitDeleteCmd)

	commitHistoryCmd.Flags().StringVar(&commitHistoryFormat, "format", "", "Output format (json)")
	commitCmd.AddCommand(commitHistoryCmd)
	commitCmd.AddCommand(commitPublishCmd)
	commitCmd.AddCommand(commitUnpublishCmd)
}
