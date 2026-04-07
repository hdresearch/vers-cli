package handlers

import (
	"context"
	"fmt"

	"github.com/hdresearch/vers-cli/internal/app"
	"github.com/hdresearch/vers-cli/internal/utils"
	vers "github.com/hdresearch/vers-sdk-go"
	"github.com/hdresearch/vers-sdk-go/option"
)

type CommitCreateReq struct {
	Target      string
	Name        string
	Description string
}

type CommitCreateView struct {
	CommitID    string `json:"commit_id"`
	VmID        string `json:"vm_id"`
	UsedHEAD    bool   `json:"used_head,omitempty"`
	Name        string `json:"name,omitempty"`
	Description string `json:"description,omitempty"`
}

func HandleCommitCreate(ctx context.Context, a *app.App, r CommitCreateReq) (CommitCreateView, error) {
	resolved, err := utils.ResolveTargetVM(ctx, a.Client, r.Target)
	if err != nil {
		return CommitCreateView{}, err
	}

	// Build request options to send name/description in the request body
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

	return CommitCreateView{
		CommitID:    resp.CommitID,
		VmID:        resolved.ID,
		UsedHEAD:    resolved.UsedHEAD,
		Name:        r.Name,
		Description: r.Description,
	}, nil
}
