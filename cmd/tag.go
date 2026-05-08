package cmd

import (
	"context"
	"fmt"

	"github.com/hdresearch/vers-cli/internal/handlers"
	pres "github.com/hdresearch/vers-cli/internal/presenters"
	"github.com/spf13/cobra"
)

var tagCmd = &cobra.Command{
	Use:   "tag",
	Short: "Manage commit tags",
	Long: `Create, list, update, and delete named tags that point to commits.
Tags provide human-readable names for commits (e.g. "production", "stable", "v1.2").`,
}

var tagCreateDescription string

var tagCreateCmd = &cobra.Command{
	Use:   "create <tag-name> <commit-id>",
	Short: "Create a new tag pointing to a commit",
	Long:  `Create a named tag that points to a specific commit. Tag names must be alphanumeric with hyphens, underscores, or dots (1-64 chars).`,
	Args:  cobra.ExactArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		apiCtx, cancel := context.WithTimeout(context.Background(), application.Timeouts.APIMedium)
		defer cancel()

		resp, err := handlers.HandleTagCreate(apiCtx, application, handlers.TagCreateReq{
			TagName:     args[0],
			CommitID:    args[1],
			Description: tagCreateDescription,
		})
		if err != nil {
			return err
		}
		fmt.Printf("Tag '%s' created -> %s\n", resp.TagName, resp.CommitID)
		return nil
	},
}

var (
	tagListQuiet  bool
	tagListJSON   bool
	tagListFormat string
	tagListLimit  int
	tagListOffset int
)

var tagListCmd = &cobra.Command{
	Use:   "list",
	Short: "List all tags",
	Long: `List all commit tags in your organization.

Use -q/--quiet to output just tag names (one per line), useful for scripting:
  vers tag delete $(vers tag list -q)   # delete all tags

Use --json for machine-readable output.

Pagination:
  --limit N    Cap results at N (default 50). Use 0 for unbounded.
  --offset N   Skip the first N results (use with --limit to page).`,
	Args: cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		apiCtx, cancel := context.WithTimeout(context.Background(), application.Timeouts.APIMedium)
		defer cancel()

		res, err := handlers.HandleTagList(apiCtx, application, handlers.TagListReq{})
		if err != nil {
			return err
		}

		format, err := pres.ParseFormat(tagListQuiet, tagListJSON, tagListFormat)
		if err != nil {
			return err
		}

		// TODO: plumb limit/offset to the SDK once server-side pagination is
		// exposed; today we trim client-side after the full response.
		start, end, info := pres.ApplyPaging(len(res.Tags), tagListLimit, tagListOffset)
		paged := res.Tags[start:end]

		switch format {
		case pres.FormatQuiet:
			names := make([]string, len(paged))
			for i, t := range paged {
				names[i] = t.TagName
			}
			pres.PrintQuiet(names)
			pres.PrintTruncationHint(cmd.ErrOrStderr(), info)
		case pres.FormatJSON:
			pres.PrintListJSON(paged, info)
		default:
			pres.RenderTagList(application, pres.TagListView{Tags: paged})
			pres.PrintTruncationHint(cmd.ErrOrStderr(), info)
		}
		return nil
	},
}

var tagGetJSON bool
var tagGetFormat string

var tagGetCmd = &cobra.Command{
	Use:   "get <tag-name>",
	Short: "Get details of a tag",
	Long: `Show detailed information about a specific tag.

Use --json for machine-readable output.`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		apiCtx, cancel := context.WithTimeout(context.Background(), application.Timeouts.APIMedium)
		defer cancel()

		info, err := handlers.HandleTagGet(apiCtx, application, handlers.TagGetReq{
			TagName: args[0],
		})
		if err != nil {
			return err
		}

		format, err := pres.ParseFormat(false, tagGetJSON, tagGetFormat)
		if err != nil {
			return err
		}
		switch format {
		case pres.FormatJSON:
			pres.PrintJSON(info)
		default:
			pres.RenderTagInfo(application, info)
		}
		return nil
	},
}

var (
	tagUpdateCommit      string
	tagUpdateDescription string
)

var tagUpdateCmd = &cobra.Command{
	Use:   "update <tag-name>",
	Short: "Update a tag",
	Long:  `Move a tag to point to a different commit, or update its description.`,
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		if tagUpdateCommit == "" && tagUpdateDescription == "" {
			return fmt.Errorf("at least one of --commit or --description must be provided")
		}

		apiCtx, cancel := context.WithTimeout(context.Background(), application.Timeouts.APIMedium)
		defer cancel()

		err := handlers.HandleTagUpdate(apiCtx, application, handlers.TagUpdateReq{
			TagName:     args[0],
			CommitID:    tagUpdateCommit,
			Description: tagUpdateDescription,
		})
		if err != nil {
			return err
		}
		fmt.Printf("Tag '%s' updated\n", args[0])
		return nil
	},
}

var tagDeleteCmd = &cobra.Command{
	Use:   "delete <tag-name>...",
	Short: "Delete one or more tags",
	Long: `Delete one or more named tags. This does not delete the commits they point to.

Examples:
  vers tag delete staging
  vers tag delete staging production
  vers tag delete $(vers tag list -q)   # delete all tags`,
	Args: cobra.MinimumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		apiCtx, cancel := context.WithTimeout(context.Background(), application.Timeouts.APIMedium)
		defer cancel()

		var firstErr error
		for _, name := range args {
			err := handlers.HandleTagDelete(apiCtx, application, handlers.TagDeleteReq{
				TagName: name,
			})
			if err != nil {
				fmt.Fprintf(cmd.ErrOrStderr(), "error: failed to delete tag '%s': %v\n", name, err)
				if firstErr == nil {
					firstErr = err
				}
				continue
			}
			fmt.Printf("Tag '%s' deleted\n", name)
		}
		return firstErr
	},
}

func init() {
	rootCmd.AddCommand(tagCmd)

	tagCreateCmd.Flags().StringVarP(&tagCreateDescription, "description", "d", "", "Description for the tag")
	tagCmd.AddCommand(tagCreateCmd)

	tagListCmd.Flags().BoolVarP(&tagListQuiet, "quiet", "q", false, "Only display tag names")
	tagListCmd.Flags().BoolVar(&tagListJSON, "json", false, "Output as JSON")
	tagListCmd.Flags().StringVar(&tagListFormat, "format", "", "Output format (json) [deprecated: use --json]")
	_ = tagListCmd.Flags().MarkDeprecated("format", "use --json instead")
	tagListCmd.Flags().IntVar(&tagListLimit, "limit", 50, "Maximum number of tags to return (0 = unbounded)")
	tagListCmd.Flags().IntVar(&tagListOffset, "offset", 0, "Number of tags to skip (for paging)")
	tagCmd.AddCommand(tagListCmd)

	tagGetCmd.Flags().BoolVar(&tagGetJSON, "json", false, "Output as JSON")
	tagGetCmd.Flags().StringVar(&tagGetFormat, "format", "", "Output format (json) [deprecated: use --json]")
	_ = tagGetCmd.Flags().MarkDeprecated("format", "use --json instead")
	tagCmd.AddCommand(tagGetCmd)

	tagUpdateCmd.Flags().StringVar(&tagUpdateCommit, "commit", "", "Move tag to this commit ID")
	tagUpdateCmd.Flags().StringVarP(&tagUpdateDescription, "description", "d", "", "New description for the tag")
	tagCmd.AddCommand(tagUpdateCmd)

	tagCmd.AddCommand(tagDeleteCmd)
}
