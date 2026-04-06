package handlers

import (
	"context"
	"fmt"

	"github.com/hdresearch/vers-cli/internal/app"
	vers "github.com/hdresearch/vers-sdk-go"
)

// ── Repo Handlers ────────────────────────────────────────────────────

type RepoCreateReq struct {
	Name        string
	Description string
}

func HandleRepoCreate(ctx context.Context, a *app.App, r RepoCreateReq) (*vers.CreateRepositoryResponse, error) {
	if r.Name == "" {
		return nil, fmt.Errorf("repository name is required")
	}
	params := vers.RepositoryNewParams{
		CreateRepositoryRequest: vers.CreateRepositoryRequestParam{
			Name: vers.F(r.Name),
		},
	}
	if r.Description != "" {
		params.CreateRepositoryRequest.Description = vers.F(r.Description)
	}
	resp, err := a.Client.Repositories.New(ctx, params)
	if err != nil {
		return nil, fmt.Errorf("failed to create repository '%s': %w", r.Name, err)
	}
	return resp, nil
}

type RepoListReq struct{}

func HandleRepoList(ctx context.Context, a *app.App, _ RepoListReq) (*vers.ListRepositoriesResponse, error) {
	resp, err := a.Client.Repositories.List(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to list repositories: %w", err)
	}
	return resp, nil
}

type RepoGetReq struct {
	Name string
}

func HandleRepoGet(ctx context.Context, a *app.App, r RepoGetReq) (*vers.RepositoryInfo, error) {
	if r.Name == "" {
		return nil, fmt.Errorf("repository name is required")
	}
	resp, err := a.Client.Repositories.Get(ctx, r.Name)
	if err != nil {
		return nil, fmt.Errorf("failed to get repository '%s': %w", r.Name, err)
	}
	return resp, nil
}

type RepoDeleteReq struct {
	Name string
}

func HandleRepoDelete(ctx context.Context, a *app.App, r RepoDeleteReq) error {
	if r.Name == "" {
		return fmt.Errorf("repository name is required")
	}
	err := a.Client.Repositories.Delete(ctx, r.Name)
	if err != nil {
		return fmt.Errorf("failed to delete repository '%s': %w", r.Name, err)
	}
	return nil
}

type RepoSetVisibilityReq struct {
	Name     string
	IsPublic bool
}

func HandleRepoSetVisibility(ctx context.Context, a *app.App, r RepoSetVisibilityReq) error {
	if r.Name == "" {
		return fmt.Errorf("repository name is required")
	}
	err := a.Client.Repositories.SetVisibility(ctx, r.Name, vers.RepositorySetVisibilityParams{
		SetRepositoryVisibilityRequest: vers.SetRepositoryVisibilityRequestParam{
			IsPublic: vers.F(r.IsPublic),
		},
	})
	if err != nil {
		return fmt.Errorf("failed to set visibility for '%s': %w", r.Name, err)
	}
	return nil
}

// ── Repo Tag Handlers ────────────────────────────────────────────────

type RepoTagCreateReq struct {
	RepoName    string
	TagName     string
	CommitID    string
	Description string
}

func HandleRepoTagCreate(ctx context.Context, a *app.App, r RepoTagCreateReq) (*vers.CreateRepoTagResponse, error) {
	if r.RepoName == "" {
		return nil, fmt.Errorf("repository name is required")
	}
	if r.TagName == "" {
		return nil, fmt.Errorf("tag name is required")
	}
	if r.CommitID == "" {
		return nil, fmt.Errorf("commit ID is required")
	}
	params := vers.RepositoryNewTagParams{
		CreateRepoTagRequest: vers.CreateRepoTagRequestParam{
			TagName:  vers.F(r.TagName),
			CommitID: vers.F(r.CommitID),
		},
	}
	if r.Description != "" {
		params.CreateRepoTagRequest.Description = vers.F(r.Description)
	}
	resp, err := a.Client.Repositories.NewTag(ctx, r.RepoName, params)
	if err != nil {
		return nil, fmt.Errorf("failed to create tag '%s' in '%s': %w", r.TagName, r.RepoName, err)
	}
	return resp, nil
}

type RepoTagListReq struct {
	RepoName string
}

func HandleRepoTagList(ctx context.Context, a *app.App, r RepoTagListReq) (*vers.ListRepoTagsResponse, error) {
	if r.RepoName == "" {
		return nil, fmt.Errorf("repository name is required")
	}
	resp, err := a.Client.Repositories.ListTags(ctx, r.RepoName)
	if err != nil {
		return nil, fmt.Errorf("failed to list tags for '%s': %w", r.RepoName, err)
	}
	return resp, nil
}

type RepoTagGetReq struct {
	RepoName string
	TagName  string
}

func HandleRepoTagGet(ctx context.Context, a *app.App, r RepoTagGetReq) (*vers.RepoTagInfo, error) {
	if r.RepoName == "" {
		return nil, fmt.Errorf("repository name is required")
	}
	if r.TagName == "" {
		return nil, fmt.Errorf("tag name is required")
	}
	resp, err := a.Client.Repositories.GetTag(ctx, r.RepoName, r.TagName)
	if err != nil {
		return nil, fmt.Errorf("failed to get tag '%s' in '%s': %w", r.TagName, r.RepoName, err)
	}
	return resp, nil
}

type RepoTagUpdateReq struct {
	RepoName    string
	TagName     string
	CommitID    string
	Description string
}

func HandleRepoTagUpdate(ctx context.Context, a *app.App, r RepoTagUpdateReq) error {
	if r.RepoName == "" {
		return fmt.Errorf("repository name is required")
	}
	if r.TagName == "" {
		return fmt.Errorf("tag name is required")
	}
	params := vers.RepositoryUpdateTagParams{
		UpdateRepoTagRequest: vers.UpdateRepoTagRequestParam{},
	}
	if r.CommitID != "" {
		params.UpdateRepoTagRequest.CommitID = vers.F(r.CommitID)
	}
	if r.Description != "" {
		params.UpdateRepoTagRequest.Description = vers.F(r.Description)
	}
	err := a.Client.Repositories.UpdateTag(ctx, r.RepoName, r.TagName, params)
	if err != nil {
		return fmt.Errorf("failed to update tag '%s' in '%s': %w", r.TagName, r.RepoName, err)
	}
	return nil
}

type RepoTagDeleteReq struct {
	RepoName string
	TagName  string
}

func HandleRepoTagDelete(ctx context.Context, a *app.App, r RepoTagDeleteReq) error {
	if r.RepoName == "" {
		return fmt.Errorf("repository name is required")
	}
	if r.TagName == "" {
		return fmt.Errorf("tag name is required")
	}
	err := a.Client.Repositories.DeleteTag(ctx, r.RepoName, r.TagName)
	if err != nil {
		return fmt.Errorf("failed to delete tag '%s' in '%s': %w", r.TagName, r.RepoName, err)
	}
	return nil
}

// ── Fork Handler ─────────────────────────────────────────────────────

type RepoForkReq struct {
	SourceOrg  string
	SourceRepo string
	SourceTag  string
	RepoName   string
	TagName    string
}

func HandleRepoFork(ctx context.Context, a *app.App, r RepoForkReq) (*vers.ForkRepositoryResponse, error) {
	if r.SourceOrg == "" {
		return nil, fmt.Errorf("source organization is required")
	}
	if r.SourceRepo == "" {
		return nil, fmt.Errorf("source repository is required")
	}
	if r.SourceTag == "" {
		return nil, fmt.Errorf("source tag is required")
	}
	params := vers.RepositoryForkParams{
		ForkRepositoryRequest: vers.ForkRepositoryRequestParam{
			SourceOrg:  vers.F(r.SourceOrg),
			SourceRepo: vers.F(r.SourceRepo),
			SourceTag:  vers.F(r.SourceTag),
		},
	}
	if r.RepoName != "" {
		params.ForkRepositoryRequest.RepoName = vers.F(r.RepoName)
	}
	if r.TagName != "" {
		params.ForkRepositoryRequest.TagName = vers.F(r.TagName)
	}
	resp, err := a.Client.Repositories.Fork(ctx, params)
	if err != nil {
		return nil, fmt.Errorf("failed to fork %s/%s:%s: %w", r.SourceOrg, r.SourceRepo, r.SourceTag, err)
	}
	return resp, nil
}
