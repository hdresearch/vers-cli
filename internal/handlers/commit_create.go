package handlers

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/hdresearch/vers-cli/internal/app"
	"github.com/hdresearch/vers-cli/internal/presenters"
	"github.com/hdresearch/vers-cli/internal/utils"
	vers "github.com/hdresearch/vers-sdk-go"
	"github.com/hdresearch/vers-sdk-go/option"
)

type CommitCreateReq struct {
	Target      string
	Name        string
	Description string
	// Tags is a list of raw "<repo>:<tag>" references to write after the
	// commit lands. Each entry creates the tag if it does not yet exist,
	// or updates an existing tag to point at the new commit.
	Tags []string
	// Public, when true, publishes the new commit (sets is_public=true)
	// after the commit and any tag writes succeed.
	Public bool
}

// CommitCreateView is re-exported from the presenters package so callers
// can keep importing it from handlers as before.
type CommitCreateView = presenters.CommitCreateView

// CommitTagWritten is re-exported for the same reason.
type CommitTagWritten = presenters.CommitTagWritten

// parsed form of a single --tag value
type tagSpec struct {
	reference string // original "repo:tag" input
	repo      string
	tag       string
}

func parseTagSpecs(raw []string) ([]tagSpec, error) {
	specs := make([]tagSpec, 0, len(raw))
	for _, s := range raw {
		parts := strings.SplitN(s, ":", 2)
		if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
			return nil, fmt.Errorf("--tag must be in <repo>:<tag> form (got: %q)", s)
		}
		specs = append(specs, tagSpec{reference: s, repo: parts[0], tag: parts[1]})
	}
	return specs, nil
}

func HandleCommitCreate(ctx context.Context, a *app.App, r CommitCreateReq) (CommitCreateView, error) {
	// 1. Validate --tag shape up-front, before any side effects.
	specs, err := parseTagSpecs(r.Tags)
	if err != nil {
		return CommitCreateView{}, err
	}

	// 2. Verify every referenced repo exists and capture its visibility.
	//    Fail-fast before creating the commit if any repo is missing.
	repoPublic := make(map[string]bool, len(specs))
	for _, s := range specs {
		if _, seen := repoPublic[s.repo]; seen {
			continue
		}
		info, err := a.Client.Repositories.Get(ctx, s.repo)
		if err != nil {
			var apiErr *vers.Error
			if errors.As(err, &apiErr) && apiErr.StatusCode == http.StatusNotFound {
				return CommitCreateView{}, fmt.Errorf("repo %q not found. Create it first with: vers repo create %s", s.repo, s.repo)
			}
			return CommitCreateView{}, fmt.Errorf("failed to look up repo %q: %w", s.repo, err)
		}
		repoPublic[s.repo] = info.IsPublic
	}

	// 3. Resolve target VM.
	resolved, err := utils.ResolveTargetVM(ctx, a.Client, r.Target)
	if err != nil {
		return CommitCreateView{}, err
	}

	// 4. Create the commit.
	var opts []option.RequestOption
	if r.Name != "" {
		opts = append(opts, option.WithJSONSet("name", r.Name))
	}
	if r.Description != "" {
		opts = append(opts, option.WithJSONSet("description", r.Description))
	}
	resp, err := a.Client.Vm.Commit(ctx, resolved.ID, vers.VmCommitParams{}, opts...)
	if err != nil {
		return CommitCreateView{}, fmt.Errorf("failed to commit VM '%s': %w", resolved.ID, err)
	}
	commitID := resp.CommitID

	view := presenters.CommitCreateView{
		CommitID:    commitID,
		VmID:        resolved.ID,
		UsedHEAD:    resolved.UsedHEAD,
		Name:        r.Name,
		Description: r.Description,
	}

	// 5. Write tags. Any failure here returns an error that names the new
	//    commit ID so the user can recover by hand.
	for _, s := range specs {
		existing, getErr := a.Client.Repositories.GetTag(ctx, s.repo, s.tag)
		var apiErr *vers.Error
		switch {
		case getErr == nil && existing != nil:
			// Tag exists -> update it to point at the new commit.
			updErr := HandleRepoTagUpdate(ctx, a, RepoTagUpdateReq{
				RepoName: s.repo,
				TagName:  s.tag,
				CommitID: commitID,
			})
			if updErr != nil {
				return view, fmt.Errorf("commit %s created, but failed to update tag %s: %w", commitID, s.reference, updErr)
			}
			view.TagsWritten = append(view.TagsWritten, presenters.CommitTagWritten{
				Reference: s.reference,
				TagID:     existing.TagID,
			})
		case errors.As(getErr, &apiErr) && apiErr.StatusCode == http.StatusNotFound:
			// Tag does not exist -> create it pointing at the new commit.
			created, createErr := HandleRepoTagCreate(ctx, a, RepoTagCreateReq{
				RepoName: s.repo,
				TagName:  s.tag,
				CommitID: commitID,
			})
			if createErr != nil {
				return view, fmt.Errorf("commit %s created, but failed to create tag %s: %w", commitID, s.reference, createErr)
			}
			view.TagsWritten = append(view.TagsWritten, presenters.CommitTagWritten{
				Reference: s.reference,
				TagID:     created.TagID,
			})
		default:
			return view, fmt.Errorf("commit %s created, but failed to look up tag %s: %w", commitID, s.reference, getErr)
		}
	}

	// 6. Publish if requested.
	if r.Public {
		info, pubErr := HandleCommitUpdate(ctx, a, CommitUpdateReq{
			CommitID: commitID,
			IsPublic: true,
		})
		if pubErr != nil {
			return view, fmt.Errorf("commit %s created (tags written: %d), but failed to publish: %w", commitID, len(view.TagsWritten), pubErr)
		}
		view.IsPublic = info.IsPublic
	}

	// 7. Visibility-mismatch warning: any tag target repo is public but
	//    the commit was not explicitly published. Stderr only — do not
	//    auto-publish (the conservative recommendation from #201).
	if !r.Public {
		var publicRepos []string
		for repo, pub := range repoPublic {
			if pub {
				publicRepos = append(publicRepos, repo)
			}
		}
		if len(publicRepos) > 0 {
			out := a.IO.Err
			if out == nil {
				out = io.Discard
			}
			fmt.Fprintf(out, "warning: commit %s was tagged into public repo(s) %s but was not published. The tag references a private commit and will not be reachable. Re-run with --public next time, or run: vers commit publish %s\n",
				commitID, strings.Join(publicRepos, ", "), commitID)
		}
	}

	return view, nil
}
