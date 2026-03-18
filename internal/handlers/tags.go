package handlers

import (
	"context"
	"fmt"

	"github.com/hdresearch/vers-cli/internal/app"
	"github.com/hdresearch/vers-cli/internal/presenters"
	vers "github.com/hdresearch/vers-sdk-go"
)

// TagCreateReq is the request for creating a tag.
type TagCreateReq struct {
	TagName     string
	CommitID    string
	Description string
}

func HandleTagCreate(ctx context.Context, a *app.App, r TagCreateReq) (*vers.CreateTagResponse, error) {
	if r.TagName == "" {
		return nil, fmt.Errorf("tag name is required")
	}
	if r.CommitID == "" {
		return nil, fmt.Errorf("commit ID is required")
	}

	params := vers.CommitTagNewParams{
		CreateTagRequest: vers.CreateTagRequestParam{
			TagName:  vers.F(r.TagName),
			CommitID: vers.F(r.CommitID),
		},
	}
	if r.Description != "" {
		params.CreateTagRequest.Description = vers.F(r.Description)
	}

	resp, err := a.Client.CommitTags.New(ctx, params)
	if err != nil {
		return nil, fmt.Errorf("failed to create tag '%s': %w", r.TagName, err)
	}
	return resp, nil
}

// TagListReq is the request for listing tags.
type TagListReq struct{}

func HandleTagList(ctx context.Context, a *app.App, r TagListReq) (presenters.TagListView, error) {
	resp, err := a.Client.CommitTags.List(ctx)
	if err != nil {
		return presenters.TagListView{}, fmt.Errorf("failed to list tags: %w", err)
	}
	if resp == nil {
		return presenters.TagListView{}, fmt.Errorf("empty response from API")
	}
	return presenters.TagListView{
		Tags: resp.Tags,
	}, nil
}

// TagGetReq is the request for getting a tag.
type TagGetReq struct {
	TagName string
}

func HandleTagGet(ctx context.Context, a *app.App, r TagGetReq) (*vers.TagInfo, error) {
	if r.TagName == "" {
		return nil, fmt.Errorf("tag name is required")
	}
	info, err := a.Client.CommitTags.Get(ctx, r.TagName)
	if err != nil {
		return nil, fmt.Errorf("failed to get tag '%s': %w", r.TagName, err)
	}
	return info, nil
}

// TagUpdateReq is the request for updating a tag.
type TagUpdateReq struct {
	TagName     string
	CommitID    string
	Description string
}

func HandleTagUpdate(ctx context.Context, a *app.App, r TagUpdateReq) error {
	if r.TagName == "" {
		return fmt.Errorf("tag name is required")
	}

	params := vers.CommitTagUpdateParams{
		UpdateTagRequest: vers.UpdateTagRequestParam{},
	}
	if r.CommitID != "" {
		params.UpdateTagRequest.CommitID = vers.F(r.CommitID)
	}
	if r.Description != "" {
		params.UpdateTagRequest.Description = vers.F(r.Description)
	}

	err := a.Client.CommitTags.Update(ctx, r.TagName, params)
	if err != nil {
		return fmt.Errorf("failed to update tag '%s': %w", r.TagName, err)
	}
	return nil
}

// TagDeleteReq is the request for deleting a tag.
type TagDeleteReq struct {
	TagName string
}

func HandleTagDelete(ctx context.Context, a *app.App, r TagDeleteReq) error {
	if r.TagName == "" {
		return fmt.Errorf("tag name is required")
	}
	err := a.Client.CommitTags.Delete(ctx, r.TagName)
	if err != nil {
		return fmt.Errorf("failed to delete tag '%s': %w", r.TagName, err)
	}
	return nil
}
