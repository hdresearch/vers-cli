package handlers

import (
	"context"
	"fmt"

	"github.com/hdresearch/vers-cli/internal/app"
	"github.com/hdresearch/vers-cli/internal/presenters"
	"github.com/hdresearch/vers-cli/internal/utils"
)

// DeployReq is the request for deploying a GitHub repo.
type DeployReq struct {
	Repo             string
	Name             string
	Branch           string
	InstallCommand   string
	BuildCommand     string
	RunCommand       string
	WorkingDirectory string
	Wait             bool
}

// deployAPIRequest matches the orchestrator's DeployRequest JSON schema.
type deployAPIRequest struct {
	Repo     string                 `json:"repo"`
	Name     *string                `json:"name,omitempty"`
	Branch   *string                `json:"branch,omitempty"`
	Settings *deploySettingsRequest `json:"settings,omitempty"`
}

// deploySettingsRequest matches the orchestrator's DeploySettings JSON schema.
type deploySettingsRequest struct {
	InstallCommand   *string `json:"install_command,omitempty"`
	BuildCommand     *string `json:"build_command,omitempty"`
	RunCommand       *string `json:"run_command,omitempty"`
	WorkingDirectory *string `json:"working_directory,omitempty"`
}

// deployAPIResponse matches the orchestrator's DeployResponse JSON schema.
type deployAPIResponse struct {
	ProjectID string `json:"project_id"`
	VmID      string `json:"vm_id"`
	Status    string `json:"status"`
}

func HandleDeploy(ctx context.Context, a *app.App, r DeployReq) (presenters.DeployView, error) {
	if r.Repo == "" {
		return presenters.DeployView{}, fmt.Errorf("repo is required (owner/repo format)")
	}

	// Build request body
	body := deployAPIRequest{
		Repo: r.Repo,
	}
	if r.Name != "" {
		body.Name = &r.Name
	}
	if r.Branch != "" {
		body.Branch = &r.Branch
	}

	// Build settings if any are specified
	if r.InstallCommand != "" || r.BuildCommand != "" || r.RunCommand != "" || r.WorkingDirectory != "" {
		settings := &deploySettingsRequest{}
		if r.InstallCommand != "" {
			settings.InstallCommand = &r.InstallCommand
		}
		if r.BuildCommand != "" {
			settings.BuildCommand = &r.BuildCommand
		}
		if r.RunCommand != "" {
			settings.RunCommand = &r.RunCommand
		}
		if r.WorkingDirectory != "" {
			settings.WorkingDirectory = &r.WorkingDirectory
		}
		body.Settings = settings
	}

	// The vers-sdk-go doesn't have a deploy service yet, so we use the
	// lower-level client.Post() to call the orchestrator endpoint directly.
	var resp deployAPIResponse
	err := a.Client.Post(ctx, "api/v1/deploy", body, &resp)
	if err != nil {
		return presenters.DeployView{}, fmt.Errorf("deploy failed: %w", err)
	}

	vmID := resp.VmID

	// Set HEAD to the newly deployed VM
	if err := utils.SetHead(vmID); err != nil {
		// Non-fatal: print a warning but don't fail the command
		fmt.Fprintf(a.IO.Err, "Warning: could not set HEAD to %s: %v\n", vmID, err)
	}

	if r.Wait {
		fmt.Fprintf(a.IO.Err, "Waiting for VM %s to be running...\n", vmID)
		if err := utils.WaitForRunning(ctx, a.Client, vmID); err != nil {
			return presenters.DeployView{}, fmt.Errorf("deploy initiated but VM failed to become running: %w", err)
		}
	}

	return presenters.DeployView{
		ProjectID: resp.ProjectID,
		VmID:      resp.VmID,
		Status:    resp.Status,
	}, nil
}


