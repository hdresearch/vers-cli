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

// DockerRunReq contains the request parameters for docker run
type DockerRunReq struct {
	DockerfilePath string
	BuildContext   string
	MemSizeMib     int64
	VcpuCount      int64
	FsSizeMib      int64
	VMAlias        string
	Detach         bool
	PortMappings   []string
	EnvVars        []string
	Interactive    bool
}

// HandleDockerRun handles the vers docker run command
func HandleDockerRun(ctx context.Context, a *app.App, req DockerRunReq) (presenters.DockerRunView, error) {
	view := presenters.DockerRunView{}

	// Validate and normalize paths
	dockerfilePath := req.DockerfilePath
	if dockerfilePath == "" {
		dockerfilePath = "Dockerfile"
	}

	// Make path absolute
	if !filepath.IsAbs(dockerfilePath) {
		cwd, err := os.Getwd()
		if err != nil {
			return view, fmt.Errorf("failed to get current directory: %w", err)
		}
		dockerfilePath = filepath.Join(cwd, dockerfilePath)
	}

	// Check if Dockerfile exists
	if _, err := os.Stat(dockerfilePath); os.IsNotExist(err) {
		return view, fmt.Errorf("Dockerfile not found: %s", dockerfilePath)
	}

	// Set default build context to Dockerfile directory
	buildContext := req.BuildContext
	if buildContext == "" {
		buildContext = filepath.Dir(dockerfilePath)
	}
	if !filepath.IsAbs(buildContext) {
		cwd, err := os.Getwd()
		if err != nil {
			return view, fmt.Errorf("failed to get current directory: %w", err)
		}
		buildContext = filepath.Join(cwd, buildContext)
	}

	// Set default VM resources
	memSize := req.MemSizeMib
	if memSize == 0 {
		memSize = 1024 // 1GB default
	}

	vcpuCount := req.VcpuCount
	if vcpuCount == 0 {
		vcpuCount = 2 // 2 vCPUs default
	}

	fsSize := req.FsSizeMib
	if fsSize == 0 {
		fsSize = 4096 // 4GB default
	}

	// Create executor and run
	executor := docker.NewExecutor(a)

	cfg := docker.RunConfig{
		DockerfilePath: dockerfilePath,
		BuildContext:   buildContext,
		MemSizeMib:     memSize,
		VcpuCount:      vcpuCount,
		FsSizeMib:      fsSize,
		VMAlias:        req.VMAlias,
		Detach:         req.Detach,
		PortMappings:   req.PortMappings,
		EnvVars:        req.EnvVars,
		Interactive:    req.Interactive,
	}

	result, err := executor.Run(ctx, cfg, a.IO.Out, a.IO.Err)
	if err != nil {
		return view, err
	}

	view.VMID = result.VMID
	view.VMAlias = result.VMAlias
	view.BaseImage = result.Dockerfile.BaseImage
	view.ExposedPorts = result.ExposedPorts
	view.StartCommand = result.StartCommand
	view.SetupComplete = result.SetupComplete
	view.Running = result.Running

	return view, nil
}
