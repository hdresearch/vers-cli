package handlers

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"github.com/hdresearch/vers-cli/internal/app"
	"github.com/hdresearch/vers-cli/internal/docker"
	"github.com/hdresearch/vers-cli/internal/presenters"
)

// DockerComposeUpReq contains the request parameters for docker compose up
type DockerComposeUpReq struct {
	ComposePath string   // Path to docker-compose.yml
	ProjectName string   // Project name (defaults to directory name)
	Detach      bool     // Run in detached mode
	Services    []string // Specific services to run (empty = all)
	NoDeps      bool     // Don't start dependencies
	EnvVars     []string // Additional environment variables
}

// HandleDockerComposeUp handles the vers docker compose up command
func HandleDockerComposeUp(ctx context.Context, a *app.App, req DockerComposeUpReq) (presenters.DockerComposeView, error) {
	view := presenters.DockerComposeView{}

	// Find compose file
	composePath := req.ComposePath
	if composePath == "" {
		// Look for common compose file names
		candidates := []string{
			"docker-compose.yml",
			"docker-compose.yaml",
			"compose.yml",
			"compose.yaml",
		}
		for _, c := range candidates {
			if _, err := os.Stat(c); err == nil {
				composePath = c
				break
			}
		}
		if composePath == "" {
			return view, fmt.Errorf("no compose file found (tried docker-compose.yml, compose.yml)")
		}
	}

	// Make path absolute
	if !filepath.IsAbs(composePath) {
		cwd, err := os.Getwd()
		if err != nil {
			return view, fmt.Errorf("failed to get current directory: %w", err)
		}
		composePath = filepath.Join(cwd, composePath)
	}

	// Check if file exists
	if _, err := os.Stat(composePath); os.IsNotExist(err) {
		return view, fmt.Errorf("compose file not found: %s", composePath)
	}

	// Determine project name
	projectName := req.ProjectName
	if projectName == "" {
		projectName = filepath.Base(filepath.Dir(composePath))
	}

	// Create executor and run
	executor := docker.NewComposeExecutor(a)

	cfg := docker.ComposeConfig{
		ComposePath: composePath,
		ProjectName: projectName,
		Detach:      req.Detach,
		Services:    req.Services,
		NoDeps:      req.NoDeps,
		EnvVars:     req.EnvVars,
	}

	result, err := executor.Up(ctx, cfg, a.IO.Out, a.IO.Err)
	if err != nil {
		return view, err
	}

	// Convert result to view
	view.ProjectName = result.ProjectName
	view.TotalServices = result.TotalServices

	for _, svc := range result.Services {
		svcView := presenters.ComposeServiceView{
			Name:          svc.Name,
			VMID:          svc.VMID,
			VMAlias:       svc.VMAlias,
			Ports:         svc.Ports,
			Running:       svc.Running,
			SetupComplete: svc.SetupComplete,
		}
		if svc.Error != nil {
			svcView.Error = svc.Error.Error()
		}
		view.Services = append(view.Services, svcView)
	}

	return view, nil
}

// DockerComposePsReq contains the request parameters for docker compose ps
type DockerComposePsReq struct {
	ProjectName string
}

// DockerComposeDownReq contains the request parameters for docker compose down
type DockerComposeDownReq struct {
	ProjectName string
	RemoveAll   bool // Also remove volumes
}
