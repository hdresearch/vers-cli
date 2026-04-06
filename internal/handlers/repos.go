package handlers

import (
	"context"
	"fmt"
	"net/http"

	"github.com/hdresearch/vers-cli/internal/app"
	pres "github.com/hdresearch/vers-cli/internal/presenters"
)

// ── DTOs (until the SDK adds repository support) ─────────────────────

type createRepoResponse struct {
	RepoID string `json:"repo_id"`
	Name   string `json:"name"`
}

type listReposResponse struct {
	Repositories []pres.RepoInfo `json:"repositories"`
}

type listRepoTagsResponse struct {
	Repository string             `json:"repository"`
	Tags       []pres.RepoTagInfo `json:"tags"`
}

type createRepoTagResponse struct {
	TagID     string `json:"tag_id"`
	Reference string `json:"reference"`
	CommitID  string `json:"commit_id"`
}

type setVisibilityRequest struct {
	IsPublic bool `json:"is_public"`
}

type forkRepoResponse struct {
	VmID      string `json:"vm_id"`
	CommitID  string `json:"commit_id"`
	RepoName  string `json:"repo_name"`
	TagName   string `json:"tag_name"`
	Reference string `json:"reference"`
}

// ── Handlers ─────────────────────────────────────────────────────────

// RepoCreateReq is the request for creating a repository.
type RepoCreateReq struct {
	Name        string
	Description string
}

func HandleRepoCreate(ctx context.Context, a *app.App, r RepoCreateReq) (*createRepoResponse, error) {
	if r.Name == "" {
		return nil, fmt.Errorf("repository name is required")
	}
	body := map[string]interface{}{"name": r.Name}
	if r.Description != "" {
		body["description"] = r.Description
	}
	var res createRepoResponse
	err := a.Client.Post(ctx, "api/v1/repositories", body, &res)
	if err != nil {
		return nil, fmt.Errorf("failed to create repository '%s': %w", r.Name, err)
	}
	return &res, nil
}

// RepoListReq is the request for listing repositories.
type RepoListReq struct{}

func HandleRepoList(ctx context.Context, a *app.App, _ RepoListReq) (*listReposResponse, error) {
	var res listReposResponse
	err := a.Client.Get(ctx, "api/v1/repositories", nil, &res)
	if err != nil {
		return nil, fmt.Errorf("failed to list repositories: %w", err)
	}
	return &res, nil
}

// RepoGetReq is the request for getting a repository.
type RepoGetReq struct {
	Name string
}

func HandleRepoGet(ctx context.Context, a *app.App, r RepoGetReq) (*pres.RepoInfo, error) {
	if r.Name == "" {
		return nil, fmt.Errorf("repository name is required")
	}
	var res pres.RepoInfo
	err := a.Client.Get(ctx, fmt.Sprintf("api/v1/repositories/%s", r.Name), nil, &res)
	if err != nil {
		return nil, fmt.Errorf("failed to get repository '%s': %w", r.Name, err)
	}
	return &res, nil
}

// RepoDeleteReq is the request for deleting a repository.
type RepoDeleteReq struct {
	Name string
}

func HandleRepoDelete(ctx context.Context, a *app.App, r RepoDeleteReq) error {
	if r.Name == "" {
		return fmt.Errorf("repository name is required")
	}
	err := a.Client.Delete(ctx, fmt.Sprintf("api/v1/repositories/%s", r.Name), nil, nil)
	if err != nil {
		return fmt.Errorf("failed to delete repository '%s': %w", r.Name, err)
	}
	return nil
}

// RepoSetVisibilityReq is the request for setting repository visibility.
type RepoSetVisibilityReq struct {
	Name     string
	IsPublic bool
}

func HandleRepoSetVisibility(ctx context.Context, a *app.App, r RepoSetVisibilityReq) error {
	if r.Name == "" {
		return fmt.Errorf("repository name is required")
	}
	body := setVisibilityRequest{IsPublic: r.IsPublic}
	err := a.Client.Execute(ctx, http.MethodPut, fmt.Sprintf("api/v1/repositories/%s/visibility", r.Name), body, nil)
	if err != nil {
		return fmt.Errorf("failed to set visibility for '%s': %w", r.Name, err)
	}
	return nil
}

// ── Repo Tag Handlers ────────────────────────────────────────────────

// RepoTagCreateReq is the request for creating a tag in a repository.
type RepoTagCreateReq struct {
	RepoName    string
	TagName     string
	CommitID    string
	Description string
}

func HandleRepoTagCreate(ctx context.Context, a *app.App, r RepoTagCreateReq) (*createRepoTagResponse, error) {
	if r.RepoName == "" {
		return nil, fmt.Errorf("repository name is required")
	}
	if r.TagName == "" {
		return nil, fmt.Errorf("tag name is required")
	}
	if r.CommitID == "" {
		return nil, fmt.Errorf("commit ID is required")
	}
	body := map[string]interface{}{
		"tag_name":  r.TagName,
		"commit_id": r.CommitID,
	}
	if r.Description != "" {
		body["description"] = r.Description
	}
	var res createRepoTagResponse
	err := a.Client.Post(ctx, fmt.Sprintf("api/v1/repositories/%s/tags", r.RepoName), body, &res)
	if err != nil {
		return nil, fmt.Errorf("failed to create tag '%s' in '%s': %w", r.TagName, r.RepoName, err)
	}
	return &res, nil
}

// RepoTagListReq is the request for listing tags in a repository.
type RepoTagListReq struct {
	RepoName string
}

func HandleRepoTagList(ctx context.Context, a *app.App, r RepoTagListReq) (*listRepoTagsResponse, error) {
	if r.RepoName == "" {
		return nil, fmt.Errorf("repository name is required")
	}
	var res listRepoTagsResponse
	err := a.Client.Get(ctx, fmt.Sprintf("api/v1/repositories/%s/tags", r.RepoName), nil, &res)
	if err != nil {
		return nil, fmt.Errorf("failed to list tags for '%s': %w", r.RepoName, err)
	}
	return &res, nil
}

// RepoTagGetReq is the request for getting a tag in a repository.
type RepoTagGetReq struct {
	RepoName string
	TagName  string
}

func HandleRepoTagGet(ctx context.Context, a *app.App, r RepoTagGetReq) (*pres.RepoTagInfo, error) {
	if r.RepoName == "" {
		return nil, fmt.Errorf("repository name is required")
	}
	if r.TagName == "" {
		return nil, fmt.Errorf("tag name is required")
	}
	var res pres.RepoTagInfo
	err := a.Client.Get(ctx, fmt.Sprintf("api/v1/repositories/%s/tags/%s", r.RepoName, r.TagName), nil, &res)
	if err != nil {
		return nil, fmt.Errorf("failed to get tag '%s' in '%s': %w", r.TagName, r.RepoName, err)
	}
	return &res, nil
}

// RepoTagUpdateReq is the request for updating a tag in a repository.
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
	body := map[string]interface{}{}
	if r.CommitID != "" {
		body["commit_id"] = r.CommitID
	}
	if r.Description != "" {
		body["description"] = r.Description
	}
	err := a.Client.Execute(ctx, http.MethodPut, fmt.Sprintf("api/v1/repositories/%s/tags/%s", r.RepoName, r.TagName), body, nil)
	if err != nil {
		return fmt.Errorf("failed to update tag '%s' in '%s': %w", r.TagName, r.RepoName, err)
	}
	return nil
}

// RepoTagDeleteReq is the request for deleting a tag in a repository.
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
	err := a.Client.Delete(ctx, fmt.Sprintf("api/v1/repositories/%s/tags/%s", r.RepoName, r.TagName), nil, nil)
	if err != nil {
		return fmt.Errorf("failed to delete tag '%s' in '%s': %w", r.TagName, r.RepoName, err)
	}
	return nil
}

// ── Fork Handler ─────────────────────────────────────────────────────

// RepoForkReq is the request for forking a public repository.
type RepoForkReq struct {
	SourceOrg  string
	SourceRepo string
	SourceTag  string
	RepoName   string
	TagName    string
}

func HandleRepoFork(ctx context.Context, a *app.App, r RepoForkReq) (*forkRepoResponse, error) {
	if r.SourceOrg == "" {
		return nil, fmt.Errorf("source organization is required")
	}
	if r.SourceRepo == "" {
		return nil, fmt.Errorf("source repository is required")
	}
	if r.SourceTag == "" {
		return nil, fmt.Errorf("source tag is required")
	}
	body := map[string]interface{}{
		"source_org":  r.SourceOrg,
		"source_repo": r.SourceRepo,
		"source_tag":  r.SourceTag,
	}
	if r.RepoName != "" {
		body["repo_name"] = r.RepoName
	}
	if r.TagName != "" {
		body["tag_name"] = r.TagName
	}
	var res forkRepoResponse
	err := a.Client.Post(ctx, "api/v1/repositories/fork", body, &res)
	if err != nil {
		return nil, fmt.Errorf("failed to fork %s/%s:%s: %w", r.SourceOrg, r.SourceRepo, r.SourceTag, err)
	}
	return &res, nil
}
