package cmd

import (
	"context"
	"fmt"

	"github.com/hdresearch/vers-cli/internal/handlers"
	pres "github.com/hdresearch/vers-cli/internal/presenters"
	"github.com/spf13/cobra"
)

var repoCmd = &cobra.Command{
	Use:   "repo",
	Short: "Manage repositories",
	Long: `Create, list, and manage repositories and their tags.
Repositories group related commits with named tags (e.g. "my-app:latest").`,
}

// ── repo create ──────────────────────────────────────────────────────

var repoCreateDescription string

var repoCreateCmd = &cobra.Command{
	Use:   "create <name>",
	Short: "Create a new repository",
	Long:  `Create a named repository. Names must be alphanumeric with hyphens, underscores, or dots (1-64 chars).`,
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		apiCtx, cancel := context.WithTimeout(context.Background(), application.Timeouts.APIMedium)
		defer cancel()

		resp, err := handlers.HandleRepoCreate(apiCtx, application, handlers.RepoCreateReq{
			Name:        args[0],
			Description: repoCreateDescription,
		})
		if err != nil {
			return err
		}
		fmt.Printf("✓ Repository '%s' created (%s)\n", resp.Name, resp.RepoID)
		return nil
	},
}

// ── repo list ────────────────────────────────────────────────────────

var (
	repoListQuiet  bool
	repoListFormat string
)

var repoListCmd = &cobra.Command{
	Use:   "list",
	Short: "List all repositories",
	Long: `List all repositories in your organization.

Use -q/--quiet to output just names (one per line), useful for scripting.
Use --format json for machine-readable output.`,
	Args: cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		apiCtx, cancel := context.WithTimeout(context.Background(), application.Timeouts.APIMedium)
		defer cancel()

		res, err := handlers.HandleRepoList(apiCtx, application, handlers.RepoListReq{})
		if err != nil {
			return err
		}

		format := pres.ParseFormat(repoListQuiet, repoListFormat)
		switch format {
		case pres.FormatQuiet:
			names := make([]string, len(res.Repositories))
			for i, r := range res.Repositories {
				names[i] = r.Name
			}
			pres.PrintQuiet(names)
		case pres.FormatJSON:
			pres.PrintJSON(res.Repositories)
		default:
			pres.RenderRepoList(application, pres.RepoListView{Repositories: res.Repositories})
		}
		return nil
	},
}

// ── repo get ─────────────────────────────────────────────────────────

var repoGetFormat string

var repoGetCmd = &cobra.Command{
	Use:   "get <name>",
	Short: "Get details of a repository",
	Long: `Show detailed information about a specific repository.

Use --format json for machine-readable output.`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		apiCtx, cancel := context.WithTimeout(context.Background(), application.Timeouts.APIMedium)
		defer cancel()

		info, err := handlers.HandleRepoGet(apiCtx, application, handlers.RepoGetReq{
			Name: args[0],
		})
		if err != nil {
			return err
		}

		format := pres.ParseFormat(false, repoGetFormat)
		switch format {
		case pres.FormatJSON:
			pres.PrintJSON(info)
		default:
			pres.RenderRepoInfo(application, info)
		}
		return nil
	},
}

// ── repo delete ──────────────────────────────────────────────────────

var repoDeleteCmd = &cobra.Command{
	Use:   "delete <name>...",
	Short: "Delete one or more repositories",
	Long: `Delete one or more repositories. This also deletes all tags within those repositories.
The commits themselves are NOT deleted.

Examples:
  vers repo delete my-app
  vers repo delete my-app staging-env
  vers repo delete $(vers repo list -q)   # delete all repos`,
	Args: cobra.MinimumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		apiCtx, cancel := context.WithTimeout(context.Background(), application.Timeouts.APIMedium)
		defer cancel()

		var firstErr error
		for _, name := range args {
			err := handlers.HandleRepoDelete(apiCtx, application, handlers.RepoDeleteReq{
				Name: name,
			})
			if err != nil {
				fmt.Fprintf(cmd.ErrOrStderr(), "✗ Failed to delete repository '%s': %v\n", name, err)
				if firstErr == nil {
					firstErr = err
				}
				continue
			}
			fmt.Printf("✓ Repository '%s' deleted\n", name)
		}
		return firstErr
	},
}

// ── repo visibility ──────────────────────────────────────────────────

var repoVisibilityPublic bool

var repoVisibilityCmd = &cobra.Command{
	Use:   "visibility <name>",
	Short: "Set repository visibility",
	Long: `Set a repository's visibility to public or private.

Examples:
  vers repo visibility my-app --public        # make public
  vers repo visibility my-app --public=false   # make private`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		apiCtx, cancel := context.WithTimeout(context.Background(), application.Timeouts.APIMedium)
		defer cancel()

		err := handlers.HandleRepoSetVisibility(apiCtx, application, handlers.RepoSetVisibilityReq{
			Name:     args[0],
			IsPublic: repoVisibilityPublic,
		})
		if err != nil {
			return err
		}

		vis := "private"
		if repoVisibilityPublic {
			vis = "public"
		}
		fmt.Printf("✓ Repository '%s' is now %s\n", args[0], vis)
		return nil
	},
}

// ── repo fork ────────────────────────────────────────────────────────

var (
	repoForkRepoName string
	repoForkTagName  string
)

var repoForkCmd = &cobra.Command{
	Use:   "fork <org>/<repo>:<tag>",
	Short: "Fork a public repository",
	Long: `Fork a public repository into your organization. Creates a new VM, commits it,
and creates a repository with a tag pointing to the commit.

Examples:
  vers repo fork acme/ubuntu:latest
  vers repo fork acme/ubuntu:latest --repo-name my-ubuntu --tag-name v1`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		org, repo, tag, err := parseRepoRef(args[0])
		if err != nil {
			return err
		}

		apiCtx, cancel := context.WithTimeout(context.Background(), application.Timeouts.APILong)
		defer cancel()

		resp, err := handlers.HandleRepoFork(apiCtx, application, handlers.RepoForkReq{
			SourceOrg:  org,
			SourceRepo: repo,
			SourceTag:  tag,
			RepoName:   repoForkRepoName,
			TagName:    repoForkTagName,
		})
		if err != nil {
			return err
		}
		fmt.Printf("✓ Forked → %s\n", resp.Reference)
		fmt.Printf("  VM:     %s\n", resp.VmID)
		fmt.Printf("  Commit: %s\n", resp.CommitID)
		return nil
	},
}

// ── repo tag (subcommand group) ──────────────────────────────────────

var repoTagCmd = &cobra.Command{
	Use:   "tag",
	Short: "Manage repository tags",
	Long:  `Create, list, update, and delete tags within a repository.`,
}

var repoTagCreateDescription string

var repoTagCreateCmd = &cobra.Command{
	Use:   "create <repo-name> <tag-name> <commit-id>",
	Short: "Create a tag in a repository",
	Long:  `Create a named tag within a repository that points to a specific commit.`,
	Args:  cobra.ExactArgs(3),
	RunE: func(cmd *cobra.Command, args []string) error {
		apiCtx, cancel := context.WithTimeout(context.Background(), application.Timeouts.APIMedium)
		defer cancel()

		resp, err := handlers.HandleRepoTagCreate(apiCtx, application, handlers.RepoTagCreateReq{
			RepoName:    args[0],
			TagName:     args[1],
			CommitID:    args[2],
			Description: repoTagCreateDescription,
		})
		if err != nil {
			return err
		}
		fmt.Printf("✓ Tag created → %s\n", resp.Reference)
		return nil
	},
}

var (
	repoTagListQuiet  bool
	repoTagListFormat string
)

var repoTagListCmd = &cobra.Command{
	Use:   "list <repo-name>",
	Short: "List tags in a repository",
	Long: `List all tags within a repository.

Use -q/--quiet for just tag names. Use --format json for machine-readable output.`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		apiCtx, cancel := context.WithTimeout(context.Background(), application.Timeouts.APIMedium)
		defer cancel()

		res, err := handlers.HandleRepoTagList(apiCtx, application, handlers.RepoTagListReq{
			RepoName: args[0],
		})
		if err != nil {
			return err
		}

		format := pres.ParseFormat(repoTagListQuiet, repoTagListFormat)
		switch format {
		case pres.FormatQuiet:
			names := make([]string, len(res.Tags))
			for i, t := range res.Tags {
				names[i] = t.TagName
			}
			pres.PrintQuiet(names)
		case pres.FormatJSON:
			pres.PrintJSON(res.Tags)
		default:
			pres.RenderRepoTagList(application, pres.RepoTagListView{
				Repository: res.Repository,
				Tags:       res.Tags,
			})
		}
		return nil
	},
}

var repoTagGetFormat string

var repoTagGetCmd = &cobra.Command{
	Use:   "get <repo-name> <tag-name>",
	Short: "Get details of a repository tag",
	Long: `Show detailed information about a specific tag within a repository.

Use --format json for machine-readable output.`,
	Args: cobra.ExactArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		apiCtx, cancel := context.WithTimeout(context.Background(), application.Timeouts.APIMedium)
		defer cancel()

		info, err := handlers.HandleRepoTagGet(apiCtx, application, handlers.RepoTagGetReq{
			RepoName: args[0],
			TagName:  args[1],
		})
		if err != nil {
			return err
		}

		format := pres.ParseFormat(false, repoTagGetFormat)
		switch format {
		case pres.FormatJSON:
			pres.PrintJSON(info)
		default:
			pres.RenderRepoTagInfo(application, info)
		}
		return nil
	},
}

var (
	repoTagUpdateCommit      string
	repoTagUpdateDescription string
)

var repoTagUpdateCmd = &cobra.Command{
	Use:   "update <repo-name> <tag-name>",
	Short: "Update a repository tag",
	Long:  `Move a tag to a different commit, or update its description.`,
	Args:  cobra.ExactArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		if repoTagUpdateCommit == "" && repoTagUpdateDescription == "" {
			return fmt.Errorf("at least one of --commit or --description must be provided")
		}

		apiCtx, cancel := context.WithTimeout(context.Background(), application.Timeouts.APIMedium)
		defer cancel()

		err := handlers.HandleRepoTagUpdate(apiCtx, application, handlers.RepoTagUpdateReq{
			RepoName:    args[0],
			TagName:     args[1],
			CommitID:    repoTagUpdateCommit,
			Description: repoTagUpdateDescription,
		})
		if err != nil {
			return err
		}
		fmt.Printf("✓ Tag '%s' in '%s' updated\n", args[1], args[0])
		return nil
	},
}

var repoTagDeleteCmd = &cobra.Command{
	Use:   "delete <repo-name> <tag-name>...",
	Short: "Delete one or more tags from a repository",
	Long: `Delete one or more tags from a repository. The commits are not deleted.

Examples:
  vers repo tag delete my-app staging
  vers repo tag delete my-app v1 v2 v3`,
	Args: cobra.MinimumNArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		repoName := args[0]
		tagNames := args[1:]

		apiCtx, cancel := context.WithTimeout(context.Background(), application.Timeouts.APIMedium)
		defer cancel()

		var firstErr error
		for _, name := range tagNames {
			err := handlers.HandleRepoTagDelete(apiCtx, application, handlers.RepoTagDeleteReq{
				RepoName: repoName,
				TagName:  name,
			})
			if err != nil {
				fmt.Fprintf(cmd.ErrOrStderr(), "✗ Failed to delete tag '%s': %v\n", name, err)
				if firstErr == nil {
					firstErr = err
				}
				continue
			}
			fmt.Printf("✓ Tag '%s' deleted from '%s'\n", name, repoName)
		}
		return firstErr
	},
}

// ── helpers ──────────────────────────────────────────────────────────

// parseRepoRef parses "org/repo:tag" into its components.
func parseRepoRef(ref string) (org, repo, tag string, err error) {
	// Find the org/repo split
	slashIdx := -1
	for i, c := range ref {
		if c == '/' {
			slashIdx = i
			break
		}
	}
	if slashIdx <= 0 {
		return "", "", "", fmt.Errorf("invalid reference '%s': expected format org/repo:tag", ref)
	}
	org = ref[:slashIdx]
	rest := ref[slashIdx+1:]

	// Find the repo:tag split
	colonIdx := -1
	for i, c := range rest {
		if c == ':' {
			colonIdx = i
			break
		}
	}
	if colonIdx <= 0 {
		return "", "", "", fmt.Errorf("invalid reference '%s': expected format org/repo:tag", ref)
	}
	repo = rest[:colonIdx]
	tag = rest[colonIdx+1:]

	if org == "" || repo == "" || tag == "" {
		return "", "", "", fmt.Errorf("invalid reference '%s': expected format org/repo:tag", ref)
	}
	return org, repo, tag, nil
}

// ── init ─────────────────────────────────────────────────────────────

func init() {
	rootCmd.AddCommand(repoCmd)

	// repo create
	repoCreateCmd.Flags().StringVarP(&repoCreateDescription, "description", "d", "", "Description for the repository")
	repoCmd.AddCommand(repoCreateCmd)

	// repo list
	repoListCmd.Flags().BoolVarP(&repoListQuiet, "quiet", "q", false, "Only display repository names")
	repoListCmd.Flags().StringVar(&repoListFormat, "format", "", "Output format (json)")
	repoCmd.AddCommand(repoListCmd)

	// repo get
	repoGetCmd.Flags().StringVar(&repoGetFormat, "format", "", "Output format (json)")
	repoCmd.AddCommand(repoGetCmd)

	// repo delete
	repoCmd.AddCommand(repoDeleteCmd)

	// repo visibility
	repoVisibilityCmd.Flags().BoolVar(&repoVisibilityPublic, "public", false, "Set to public (use --public=false for private)")
	repoCmd.AddCommand(repoVisibilityCmd)

	// repo fork
	repoForkCmd.Flags().StringVar(&repoForkRepoName, "repo-name", "", "Name for the forked repository (default: source name)")
	repoForkCmd.Flags().StringVar(&repoForkTagName, "tag-name", "", "Tag name in the new repo (default: source tag)")
	repoCmd.AddCommand(repoForkCmd)

	// repo tag subcommands
	repoCmd.AddCommand(repoTagCmd)

	repoTagCreateCmd.Flags().StringVarP(&repoTagCreateDescription, "description", "d", "", "Description for the tag")
	repoTagCmd.AddCommand(repoTagCreateCmd)

	repoTagListCmd.Flags().BoolVarP(&repoTagListQuiet, "quiet", "q", false, "Only display tag names")
	repoTagListCmd.Flags().StringVar(&repoTagListFormat, "format", "", "Output format (json)")
	repoTagCmd.AddCommand(repoTagListCmd)

	repoTagGetCmd.Flags().StringVar(&repoTagGetFormat, "format", "", "Output format (json)")
	repoTagCmd.AddCommand(repoTagGetCmd)

	repoTagUpdateCmd.Flags().StringVar(&repoTagUpdateCommit, "commit", "", "Move tag to this commit ID")
	repoTagUpdateCmd.Flags().StringVarP(&repoTagUpdateDescription, "description", "d", "", "New description for the tag")
	repoTagCmd.AddCommand(repoTagUpdateCmd)

	repoTagCmd.AddCommand(repoTagDeleteCmd)
}
