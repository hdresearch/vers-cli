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
		fmt.Printf("✓ Tag '%s' created → %s\n", resp.TagName, resp.CommitID)
		return nil
	},
}

var tagListCmd = &cobra.Command{
	Use:   "list",
	Short: "List all tags",
	Long:  `List all commit tags in your organization.`,
	Args:  cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		apiCtx, cancel := context.WithTimeout(context.Background(), application.Timeouts.APIMedium)
		defer cancel()

		res, err := handlers.HandleTagList(apiCtx, application, handlers.TagListReq{})
		if err != nil {
			return err
		}
		pres.RenderTagList(application, res)
		return nil
	},
}

var tagGetCmd = &cobra.Command{
	Use:   "get <tag-name>",
	Short: "Get details of a tag",
	Long:  `Show detailed information about a specific tag.`,
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		apiCtx, cancel := context.WithTimeout(context.Background(), application.Timeouts.APIMedium)
		defer cancel()

		info, err := handlers.HandleTagGet(apiCtx, application, handlers.TagGetReq{
			TagName: args[0],
		})
		if err != nil {
			return err
		}
		pres.RenderTagInfo(application, info)
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
		fmt.Printf("✓ Tag '%s' updated\n", args[0])
		return nil
	},
}

var tagDeleteCmd = &cobra.Command{
	Use:   "delete <tag-name>",
	Short: "Delete a tag",
	Long:  `Delete a named tag. This does not delete the commit it points to.`,
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		apiCtx, cancel := context.WithTimeout(context.Background(), application.Timeouts.APIMedium)
		defer cancel()

		err := handlers.HandleTagDelete(apiCtx, application, handlers.TagDeleteReq{
			TagName: args[0],
		})
		if err != nil {
			return err
		}
		fmt.Printf("✓ Tag '%s' deleted\n", args[0])
		return nil
	},
}

func init() {
	rootCmd.AddCommand(tagCmd)

	tagCreateCmd.Flags().StringVarP(&tagCreateDescription, "description", "d", "", "Description for the tag")
	tagCmd.AddCommand(tagCreateCmd)

	tagCmd.AddCommand(tagListCmd)
	tagCmd.AddCommand(tagGetCmd)

	tagUpdateCmd.Flags().StringVar(&tagUpdateCommit, "commit", "", "Move tag to this commit ID")
	tagUpdateCmd.Flags().StringVarP(&tagUpdateDescription, "description", "d", "", "New description for the tag")
	tagCmd.AddCommand(tagUpdateCmd)

	tagCmd.AddCommand(tagDeleteCmd)
}
