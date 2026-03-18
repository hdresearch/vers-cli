package handlers

import (
	"context"
	"fmt"

	"github.com/hdresearch/vers-cli/internal/app"
	"github.com/hdresearch/vers-cli/internal/presenters"
	vers "github.com/hdresearch/vers-sdk-go"
)

// CommitsListReq is the request for listing commits.
type CommitsListReq struct {
	Public bool
}

func HandleCommitsList(ctx context.Context, a *app.App, r CommitsListReq) (presenters.CommitsListView, error) {
	var resp *vers.ListCommitsResponse
	var err error

	if r.Public {
		resp, err = a.Client.Commits.ListPublic(ctx)
	} else {
		resp, err = a.Client.Commits.List(ctx)
	}
	if err != nil {
		return presenters.CommitsListView{}, fmt.Errorf("failed to list commits: %w", err)
	}
	if resp == nil {
		return presenters.CommitsListView{}, fmt.Errorf("empty response from API")
	}
	return presenters.CommitsListView{
		Commits: resp.Commits,
		Total:   resp.Total,
		Public:  r.Public,
	}, nil
}

// CommitDeleteReq is the request for deleting a commit.
type CommitDeleteReq struct {
	CommitID string
}

func HandleCommitDelete(ctx context.Context, a *app.App, r CommitDeleteReq) error {
	if r.CommitID == "" {
		return fmt.Errorf("commit ID is required")
	}
	err := a.Client.Commits.Delete(ctx, r.CommitID)
	if err != nil {
		return fmt.Errorf("failed to delete commit '%s': %w", r.CommitID, err)
	}
	return nil
}

// CommitUpdateReq is the request for updating a commit.
type CommitUpdateReq struct {
	CommitID string
	IsPublic bool
}

func HandleCommitUpdate(ctx context.Context, a *app.App, r CommitUpdateReq) (*vers.CommitInfo, error) {
	if r.CommitID == "" {
		return nil, fmt.Errorf("commit ID is required")
	}
	info, err := a.Client.Commits.Update(ctx, r.CommitID, vers.CommitUpdateParams{
		UpdateCommitRequest: vers.UpdateCommitRequestParam{
			IsPublic: vers.F(r.IsPublic),
		},
	})
	if err != nil {
		return nil, fmt.Errorf("failed to update commit '%s': %w", r.CommitID, err)
	}
	return info, nil
}

// CommitParentsReq is the request for listing parent commits.
type CommitParentsReq struct {
	CommitID string
}

func HandleCommitParents(ctx context.Context, a *app.App, r CommitParentsReq) (presenters.CommitParentsView, error) {
	if r.CommitID == "" {
		return presenters.CommitParentsView{}, fmt.Errorf("commit ID is required")
	}
	resp, err := a.Client.Commits.ListParents(ctx, r.CommitID)
	if err != nil {
		return presenters.CommitParentsView{}, fmt.Errorf("failed to list parents for commit '%s': %w", r.CommitID, err)
	}
	var parents []vers.CommitListParentsResponse
	if resp != nil {
		parents = *resp
	}
	return presenters.CommitParentsView{
		CommitID: r.CommitID,
		Parents:  parents,
	}, nil
}
