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

var (
	repoCreateDescription string
	repoCreateJSON        bool
	repoCreateFormat      string
)

var repoCreateCmd = &cobra.Command{
	Use:   "create <name>",
	Short: "Create a new repository",
	Long: `Create a named repository. Names must be alphanumeric with hyphens, underscores, or dots (1-64 chars).

Use --json for machine-readable output (returns the repository name and repo_id).`,
	Args: cobra.ExactArgs(1),
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

		format, err := pres.ParseFormat(false, repoCreateJSON, repoCreateFormat)
		if err != nil {
			return err
		}
		switch format {
		case pres.FormatJSON:
			return pres.PrintJSON(resp)
		default:
			fmt.Printf("Repository '%s' created (%s)\n", resp.Name, resp.RepoID)
		}
		return nil
	},
}

// ── repo list ────────────────────────────────────────────────────────

var (
	repoListQuiet  bool
	repoListJSON   bool
	repoListFormat string
	repoListLimit  int
	repoListOffset int
)

var repoListCmd = &cobra.Command{
	Use:   "list",
	Short: "List all repositories",
	Long: `List all repositories in your organization.

Use -q/--quiet to output just names (one per line), useful for scripting.
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

		res, err := handlers.HandleRepoList(apiCtx, application, handlers.RepoListReq{})
		if err != nil {
			return err
		}

		format, err := pres.ParseFormat(repoListQuiet, repoListJSON, repoListFormat)
		if err != nil {
			return err
		}

		// TODO: when the SDK exposes server-side limit/offset for repo list,
		// plumb repoListLimit/repoListOffset through instead of trimming here.
		start, end, info := pres.ApplyPaging(len(res.Repositories), repoListLimit, repoListOffset)
		paged := res.Repositories[start:end]

		switch format {
		case pres.FormatQuiet:
			names := make([]string, len(paged))
			for i, r := range paged {
				names[i] = r.Name
			}
			pres.PrintQuiet(names)
			pres.PrintTruncationHint(cmd.ErrOrStderr(), info)
		case pres.FormatJSON:
			pres.PrintListJSON(paged, info)
		default:
			pres.RenderRepoList(application, pres.RepoListView{Repositories: paged})
			pres.PrintTruncationHint(cmd.ErrOrStderr(), info)
		}
		return nil
	},
}

// ── repo get ─────────────────────────────────────────────────────────

var repoGetJSON bool
var repoGetFormat string

var repoGetCmd = &cobra.Command{
	Use:   "get <name>",
	Short: "Get details of a repository",
	Long: `Show detailed information about a specific repository.

Use --json for machine-readable output.`,
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

		format, err := pres.ParseFormat(false, repoGetJSON, repoGetFormat)
		if err != nil {
			return err
		}
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
				fmt.Fprintf(cmd.ErrOrStderr(), "error: failed to delete repository '%s': %v\n", name, err)
				if firstErr == nil {
					firstErr = err
				}
				continue
			}
			fmt.Printf("Repository '%s' deleted\n", name)
		}
		return firstErr
	},
}

// ── repo visibility ──────────────────────────────────────────────────

var (
	repoVisibilityPublic  bool
	repoVisibilityPrivate bool
	repoVisibilityJSON    bool
	repoVisibilityFormat  string
)

var repoVisibilityCmd = &cobra.Command{
	Use:   "visibility <name>",
	Short: "Set repository visibility",
	Long: `Set a repository's visibility to public or private.

Exactly one of --public or --private must be specified. The legacy
--public=false form is no longer accepted; use --private instead.

Use --json for machine-readable output.

Examples:
  vers repo visibility my-app --public      # make public
  vers repo visibility my-app --private     # make private`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		isPublic := repoVisibilityPublic

		apiCtx, cancel := context.WithTimeout(context.Background(), application.Timeouts.APIMedium)
		defer cancel()

		err := handlers.HandleRepoSetVisibility(apiCtx, application, handlers.RepoSetVisibilityReq{
			Name:     args[0],
			IsPublic: isPublic,
		})
		if err != nil {
			return err
		}

		format, err := pres.ParseFormat(false, repoVisibilityJSON, repoVisibilityFormat)
		if err != nil {
			return err
		}
		switch format {
		case pres.FormatJSON:
			return pres.PrintJSON(struct {
				Name     string `json:"name"`
				IsPublic bool   `json:"is_public"`
			}{Name: args[0], IsPublic: isPublic})
		default:
			vis := "private"
			if isPublic {
				vis = "public"
			}
			fmt.Printf("Repository '%s' is now %s\n", args[0], vis)
		}
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
		fmt.Printf("Forked -> %s\n", resp.Reference)
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

var (
	repoTagCreateDescription string
	repoTagCreateJSON        bool
	repoTagCreateFormat      string
)

var repoTagCreateCmd = &cobra.Command{
	Use:   "create <repo-name> <tag-name> <commit-id>",
	Short: "Create a tag in a repository",
	Long: `Create a named tag within a repository that points to a specific commit.

Use --json for machine-readable output (returns repo, tag_name, commit_id, tag_id, reference).`,
	Args: cobra.ExactArgs(3),
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

		format, err := pres.ParseFormat(false, repoTagCreateJSON, repoTagCreateFormat)
		if err != nil {
			return err
		}
		switch format {
		case pres.FormatJSON:
			return pres.PrintJSON(struct {
				Repo      string `json:"repo"`
				TagName   string `json:"tag_name"`
				CommitID  string `json:"commit_id"`
				TagID     string `json:"tag_id"`
				Reference string `json:"reference"`
			}{
				Repo:      args[0],
				TagName:   args[1],
				CommitID:  resp.CommitID,
				TagID:     resp.TagID,
				Reference: resp.Reference,
			})
		default:
			fmt.Printf("Tag created -> %s\n", resp.Reference)
		}
		return nil
	},
}

var (
	repoTagListQuiet  bool
	repoTagListJSON   bool
	repoTagListFormat string
	repoTagListLimit  int
	repoTagListOffset int
)

var repoTagListCmd = &cobra.Command{
	Use:   "list <repo-name>",
	Short: "List tags in a repository",
	Long: `List all tags within a repository.

Use -q/--quiet for just tag names. Use --json for machine-readable output.

Pagination:
  --limit N    Cap results at N (default 50). Use 0 for unbounded.
  --offset N   Skip the first N results (use with --limit to page).`,
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

		format, err := pres.ParseFormat(repoTagListQuiet, repoTagListJSON, repoTagListFormat)
		if err != nil {
			return err
		}

		// TODO: plumb limit/offset to the SDK once server-side pagination is
		// exposed; today we trim client-side after the full response.
		start, end, info := pres.ApplyPaging(len(res.Tags), repoTagListLimit, repoTagListOffset)
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
			pres.RenderRepoTagList(application, pres.RepoTagListView{
				Repository: res.Repository,
				Tags:       paged,
			})
			pres.PrintTruncationHint(cmd.ErrOrStderr(), info)
		}
		return nil
	},
}

var repoTagGetJSON bool
var repoTagGetFormat string

var repoTagGetCmd = &cobra.Command{
	Use:   "get <repo-name> <tag-name>",
	Short: "Get details of a repository tag",
	Long: `Show detailed information about a specific tag within a repository.

Use --json for machine-readable output.`,
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

		format, err := pres.ParseFormat(false, repoTagGetJSON, repoTagGetFormat)
		if err != nil {
			return err
		}
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
	repoTagUpdateJSON        bool
	repoTagUpdateFormat      string
)

var repoTagUpdateCmd = &cobra.Command{
	Use:   "update <repo-name> <tag-name>",
	Short: "Update a repository tag",
	Long: `Move a tag to a different commit, or update its description.

Use --json for machine-readable output (returns repo, tag_name, reference, and any
updated fields).`,
	Args: cobra.ExactArgs(2),
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

		format, err := pres.ParseFormat(false, repoTagUpdateJSON, repoTagUpdateFormat)
		if err != nil {
			return err
		}
		switch format {
		case pres.FormatJSON:
			return pres.PrintJSON(struct {
				Repo        string `json:"repo"`
				TagName     string `json:"tag_name"`
				Reference   string `json:"reference"`
				CommitID    string `json:"commit_id,omitempty"`
				Description string `json:"description,omitempty"`
			}{
				Repo:        args[0],
				TagName:     args[1],
				Reference:   fmt.Sprintf("%s:%s", args[0], args[1]),
				CommitID:    repoTagUpdateCommit,
				Description: repoTagUpdateDescription,
			})
		default:
			fmt.Printf("Tag '%s' in '%s' updated\n", args[1], args[0])
		}
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
				fmt.Fprintf(cmd.ErrOrStderr(), "error: failed to delete tag '%s': %v\n", name, err)
				if firstErr == nil {
					firstErr = err
				}
				continue
			}
			fmt.Printf("Tag '%s' deleted from '%s'\n", name, repoName)
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
	repoCreateCmd.Flags().BoolVar(&repoCreateJSON, "json", false, "Output as JSON")
	repoCreateCmd.Flags().StringVar(&repoCreateFormat, "format", "", "Output format (json) [deprecated: use --json]")
	_ = repoCreateCmd.Flags().MarkDeprecated("format", "use --json instead")
	repoCmd.AddCommand(repoCreateCmd)

	// repo list
	repoListCmd.Flags().BoolVarP(&repoListQuiet, "quiet", "q", false, "Only display repository names")
	repoListCmd.Flags().BoolVar(&repoListJSON, "json", false, "Output as JSON")
	repoListCmd.Flags().StringVar(&repoListFormat, "format", "", "Output format (json) [deprecated: use --json]")
	_ = repoListCmd.Flags().MarkDeprecated("format", "use --json instead")
	repoListCmd.Flags().IntVar(&repoListLimit, "limit", 50, "Maximum number of repositories to return (0 = unbounded)")
	repoListCmd.Flags().IntVar(&repoListOffset, "offset", 0, "Number of repositories to skip (for paging)")
	repoCmd.AddCommand(repoListCmd)

	// repo get
	repoGetCmd.Flags().BoolVar(&repoGetJSON, "json", false, "Output as JSON")
	repoGetCmd.Flags().StringVar(&repoGetFormat, "format", "", "Output format (json) [deprecated: use --json]")
	_ = repoGetCmd.Flags().MarkDeprecated("format", "use --json instead")
	repoCmd.AddCommand(repoGetCmd)

	// repo delete
	repoCmd.AddCommand(repoDeleteCmd)

	// repo visibility
	repoVisibilityCmd.Flags().BoolVar(&repoVisibilityPublic, "public", false, "Make the repository public")
	repoVisibilityCmd.Flags().BoolVar(&repoVisibilityPrivate, "private", false, "Make the repository private")
	repoVisibilityCmd.MarkFlagsMutuallyExclusive("public", "private")
	repoVisibilityCmd.MarkFlagsOneRequired("public", "private")
	repoVisibilityCmd.Flags().BoolVar(&repoVisibilityJSON, "json", false, "Output as JSON")
	repoVisibilityCmd.Flags().StringVar(&repoVisibilityFormat, "format", "", "Output format (json) [deprecated: use --json]")
	_ = repoVisibilityCmd.Flags().MarkDeprecated("format", "use --json instead")
	repoCmd.AddCommand(repoVisibilityCmd)

	// repo fork
	repoForkCmd.Flags().StringVar(&repoForkRepoName, "repo-name", "", "Name for the forked repository (default: source name)")
	repoForkCmd.Flags().StringVar(&repoForkTagName, "tag-name", "", "Tag name in the new repo (default: source tag)")
	repoCmd.AddCommand(repoForkCmd)

	// repo tag subcommands
	repoCmd.AddCommand(repoTagCmd)

	repoTagCreateCmd.Flags().StringVarP(&repoTagCreateDescription, "description", "d", "", "Description for the tag")
	repoTagCreateCmd.Flags().BoolVar(&repoTagCreateJSON, "json", false, "Output as JSON")
	repoTagCreateCmd.Flags().StringVar(&repoTagCreateFormat, "format", "", "Output format (json) [deprecated: use --json]")
	_ = repoTagCreateCmd.Flags().MarkDeprecated("format", "use --json instead")
	repoTagCmd.AddCommand(repoTagCreateCmd)

	repoTagListCmd.Flags().BoolVarP(&repoTagListQuiet, "quiet", "q", false, "Only display tag names")
	repoTagListCmd.Flags().BoolVar(&repoTagListJSON, "json", false, "Output as JSON")
	repoTagListCmd.Flags().StringVar(&repoTagListFormat, "format", "", "Output format (json) [deprecated: use --json]")
	_ = repoTagListCmd.Flags().MarkDeprecated("format", "use --json instead")
	repoTagListCmd.Flags().IntVar(&repoTagListLimit, "limit", 50, "Maximum number of tags to return (0 = unbounded)")
	repoTagListCmd.Flags().IntVar(&repoTagListOffset, "offset", 0, "Number of tags to skip (for paging)")
	repoTagCmd.AddCommand(repoTagListCmd)

	repoTagGetCmd.Flags().BoolVar(&repoTagGetJSON, "json", false, "Output as JSON")
	repoTagGetCmd.Flags().StringVar(&repoTagGetFormat, "format", "", "Output format (json) [deprecated: use --json]")
	_ = repoTagGetCmd.Flags().MarkDeprecated("format", "use --json instead")
	repoTagCmd.AddCommand(repoTagGetCmd)

	repoTagUpdateCmd.Flags().StringVar(&repoTagUpdateCommit, "commit", "", "Move tag to this commit ID")
	repoTagUpdateCmd.Flags().StringVarP(&repoTagUpdateDescription, "description", "d", "", "New description for the tag")
	repoTagUpdateCmd.Flags().BoolVar(&repoTagUpdateJSON, "json", false, "Output as JSON")
	repoTagUpdateCmd.Flags().StringVar(&repoTagUpdateFormat, "format", "", "Output format (json) [deprecated: use --json]")
	_ = repoTagUpdateCmd.Flags().MarkDeprecated("format", "use --json instead")
	repoTagCmd.AddCommand(repoTagUpdateCmd)

	repoTagCmd.AddCommand(repoTagDeleteCmd)
}
