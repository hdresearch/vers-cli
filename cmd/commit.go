package cmd

import (
	"context"
	"fmt"

	"github.com/hdresearch/vers-cli/internal/handlers"
	pres "github.com/hdresearch/vers-cli/internal/presenters"
	"github.com/spf13/cobra"
)

var (
	commitJSON        bool
	commitFormat      string
	commitName        string
	commitDescription string
	commitTags        []string
	commitPublic      bool
)

// commitCmd is the parent command for commit operations.
// Bare `vers commit` (no args, no subcommand) creates a commit of HEAD for backward compat.
var commitCmd = &cobra.Command{
	Use:   "commit",
	Short: "Manage commits",
	Long: `Manage commits — create, list, delete, and inspect commit history.

Create a commit (saves current VM state):
  vers commit                        # commit HEAD VM
  vers commit create [vm-id]         # commit a specific VM

Subcommands:
  create     Create a commit (default when no subcommand given)
  list       List your commits
  delete     Delete a commit
  history    Show the parent commit chain
  publish    Make a commit public
  unpublish  Make a commit private`,
	Args: cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		// Bare `vers commit` with no args -> create commit of HEAD
		return commitCreateCmd.RunE(cmd, args)
	},
}

var commitCreateCmd = &cobra.Command{
	Use:   "create [vm-id|alias]",
	Short: "Create a commit from a VM",
	Long: `Save the current state of a VM as a commit.
If no VM ID or alias is provided, commits the current HEAD VM.

Use --name to give the commit a human-readable name.
Use --description to add additional context.
Use --tag <repo>:<tag> (repeatable) to create or update repo tags pointing at the new commit.
Use --public to publish the commit (set is_public=true) after it lands.
Use --json for machine-readable output.

Examples:
  vers commit create --name "golden-image-v3"
  vers commit create --name "pre-deploy" --description "Before deploying auth changes"
  vers commit create vm-123 --name "checkpoint"
  vers commit create vm-123 --tag my-app:v1.2 --tag my-app:latest --public`,
	Args: cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		target := ""
		if len(args) > 0 {
			target = args[0]
		}

		apiCtx, cancel := context.WithTimeout(context.Background(), application.Timeouts.APILong)
		defer cancel()

		res, err := handlers.HandleCommitCreate(apiCtx, application, handlers.CommitCreateReq{
			Target:      target,
			Name:        commitName,
			Description: commitDescription,
			Tags:        commitTags,
			Public:      commitPublic,
		})
		if err != nil {
			return err
		}

		format, err := pres.ParseFormat(false, commitJSON, commitFormat)
		if err != nil {
			return err
		}
		switch format {
		case pres.FormatJSON:
			pres.PrintJSON(res)
		default:
			pres.RenderCommitCreate(application, res)
		}
		return nil
	},
}

var (
	commitListPublic bool
	commitListQuiet  bool
	commitListJSON   bool
	commitListFormat string
	commitListLimit  int
	commitListOffset int
)

var commitListCmd = &cobra.Command{
	Use:   "list",
	Short: "List your commits",
	Long: `List all commits owned by your account. Use --public to list publicly shared commits instead.

Use -q/--quiet to output just commit IDs (one per line), useful for scripting:
  vers commit delete $(vers commit list -q)   # delete all commits

Use --json for machine-readable output.

Pagination:
  --limit N    Cap results at N (default 50). Use 0 for unbounded.
  --offset N   Skip the first N results (use with --limit to page).

When the result is truncated, a hint with --offset for the next page is
printed to stderr (text mode) or included in the JSON envelope.`,
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

		format, err := pres.ParseFormat(commitListQuiet, commitListJSON, commitListFormat)
		if err != nil {
			return err
		}

		// Apply client-side pagination over res.Commits.
		// TODO: when the SDK exposes server-side limit/offset on the commit
		// list endpoint, plumb commitListLimit/commitListOffset through to
		// the API and use the server-reported total instead of len(items).
		start, end, info := pres.ApplyPaging(len(res.Commits), commitListLimit, commitListOffset)
		paged := res.Commits[start:end]

		switch format {
		case pres.FormatQuiet:
			ids := make([]string, len(paged))
			for i, c := range paged {
				ids[i] = c.CommitID
			}
			pres.PrintQuiet(ids)
			pres.PrintTruncationHint(cmd.ErrOrStderr(), info)
		case pres.FormatJSON:
			pres.PrintListJSON(paged, info)
		default:
			pagedView := res
			pagedView.Commits = paged
			pres.RenderCommitsList(application, pagedView)
			pres.PrintTruncationHint(cmd.ErrOrStderr(), info)
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
				fmt.Fprintf(cmd.ErrOrStderr(), "error: failed to delete commit %s: %v\n", id, err)
				if firstErr == nil {
					firstErr = err
				}
				continue
			}
			fmt.Printf("Commit %s deleted\n", id)
		}
		return firstErr
	},
}

var commitHistoryJSON bool
var commitHistoryFormat string

var commitHistoryCmd = &cobra.Command{
	Use:   "history <commit-id>",
	Short: "Show the parent commit chain",
	Long: `Display the chain of parent commits leading up to a given commit.

Use --json for machine-readable output.`,
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

		format, err := pres.ParseFormat(false, commitHistoryJSON, commitHistoryFormat)
		if err != nil {
			return err
		}
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
		fmt.Printf("Commit %s is now public\n", info.CommitID)
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
		fmt.Printf("Commit %s is now private\n", info.CommitID)
		return nil
	},
}

func init() {
	rootCmd.AddCommand(commitCmd)

	commitCreateCmd.Flags().BoolVar(&commitJSON, "json", false, "Output as JSON")
	commitCreateCmd.Flags().StringVar(&commitFormat, "format", "", "Output format (json) [deprecated: use --json]")
	_ = commitCreateCmd.Flags().MarkDeprecated("format", "use --json instead")
	commitCreateCmd.Flags().StringVarP(&commitName, "name", "n", "", "Human-readable name for the commit")
	commitCreateCmd.Flags().StringVarP(&commitDescription, "description", "d", "", "Description for the commit")
	commitCreateCmd.Flags().StringSliceVar(&commitTags, "tag", nil, "Repo tag to write pointing at the new commit, in <repo>:<tag> form (repeatable)")
	commitCreateCmd.Flags().BoolVar(&commitPublic, "public", false, "Publish the commit (set is_public=true) after it lands")
	commitCmd.AddCommand(commitCreateCmd)

	commitListCmd.Flags().BoolVar(&commitListPublic, "public", false, "List public commits instead of your own")
	commitListCmd.Flags().BoolVarP(&commitListQuiet, "quiet", "q", false, "Only display commit IDs")
	commitListCmd.Flags().BoolVar(&commitListJSON, "json", false, "Output as JSON")
	commitListCmd.Flags().StringVar(&commitListFormat, "format", "", "Output format (json) [deprecated: use --json]")
	_ = commitListCmd.Flags().MarkDeprecated("format", "use --json instead")
	commitListCmd.Flags().IntVar(&commitListLimit, "limit", 50, "Maximum number of commits to return (0 = unbounded)")
	commitListCmd.Flags().IntVar(&commitListOffset, "offset", 0, "Number of commits to skip (for paging)")
	commitCmd.AddCommand(commitListCmd)
	commitCmd.AddCommand(commitDeleteCmd)

	commitHistoryCmd.Flags().BoolVar(&commitHistoryJSON, "json", false, "Output as JSON")
	commitHistoryCmd.Flags().StringVar(&commitHistoryFormat, "format", "", "Output format (json) [deprecated: use --json]")
	_ = commitHistoryCmd.Flags().MarkDeprecated("format", "use --json instead")
	commitCmd.AddCommand(commitHistoryCmd)
	commitCmd.AddCommand(commitPublishCmd)
	commitCmd.AddCommand(commitUnpublishCmd)
}
