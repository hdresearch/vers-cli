package handlers

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/hdresearch/vers-cli/internal/app"
	"github.com/hdresearch/vers-cli/internal/docker"
	"github.com/hdresearch/vers-cli/internal/presenters"
	vmSvc "github.com/hdresearch/vers-cli/internal/services/vm"
	sshutil "github.com/hdresearch/vers-cli/internal/ssh"
	"github.com/hdresearch/vers-cli/internal/utils"
	vers "github.com/hdresearch/vers-sdk-go"
)

// DockerBuildReq contains the request parameters for docker build
type DockerBuildReq struct {
	DockerfilePath string
	BuildContext   string
	Tag            string // Tag/name for the commit
	MemSizeMib     int64
	VcpuCount      int64
	FsSizeMib      int64
	NoCache        bool
	BuildArgs      []string // Build arguments
}

// HandleDockerBuild handles the vers docker build command
// It builds a Dockerfile and creates a Vers commit (snapshot)
func HandleDockerBuild(ctx context.Context, a *app.App, req DockerBuildReq) (presenters.DockerBuildView, error) {
	view := presenters.DockerBuildView{}

	// Validate and normalize paths
	dockerfilePath := req.DockerfilePath
	if dockerfilePath == "" {
		dockerfilePath = "Dockerfile"
	}

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

	// Set default build context
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

	// Set defaults
	memSize := req.MemSizeMib
	if memSize == 0 {
		memSize = 1024
	}
	vcpuCount := req.VcpuCount
	if vcpuCount == 0 {
		vcpuCount = 2
	}
	fsSize := req.FsSizeMib
	if fsSize == 0 {
		fsSize = 4096
	}

	// Parse Dockerfile
	df, err := docker.ParseDockerfile(dockerfilePath)
	if err != nil {
		return view, fmt.Errorf("failed to parse Dockerfile: %w", err)
	}

	view.BaseImage = df.BaseImage
	view.NumLayers = len(df.GetRunCommands())

	fmt.Fprintln(a.IO.Out, "📄 Parsed Dockerfile")
	fmt.Fprintf(a.IO.Out, "   Base image: %s\n", df.BaseImage)
	fmt.Fprintf(a.IO.Out, "   Layers: %d RUN commands\n", view.NumLayers)

	// Step 1: Create VM
	fmt.Fprintln(a.IO.Out, "\n🚀 Creating build VM...")

	vmConfig := vers.NewRootRequestVmConfigParam{
		MemSizeMib: vers.F(memSize),
		VcpuCount:  vers.F(vcpuCount),
		FsSizeMib:  vers.F(fsSize),
	}

	body := vers.VmNewRootParams{
		NewRootRequest: vers.NewRootRequestParam{
			VmConfig: vers.F(vmConfig),
		},
	}

	resp, err := a.Client.Vm.NewRoot(ctx, body)
	if err != nil {
		return view, fmt.Errorf("failed to create VM: %w", err)
	}

	view.VMID = resp.VmID
	fmt.Fprintf(a.IO.Out, "   VM: %s\n", view.VMID)

	// Step 2: Get SSH connection
	info, err := vmSvc.GetConnectInfo(ctx, a.Client, view.VMID)
	if err != nil {
		return view, fmt.Errorf("failed to get VM connection info: %w", err)
	}

	sshClient := sshutil.NewClient(info.Host, info.KeyPath, info.VMDomain)

	// Wait for VM
	fmt.Fprintln(a.IO.Out, "\n⏳ Waiting for VM...")
	if err := waitForSSH(ctx, sshClient, a.IO.Out); err != nil {
		return view, fmt.Errorf("VM not ready: %w", err)
	}

	// Step 3: Setup workdir
	workdir := df.WorkDir
	if workdir == "" {
		workdir = "/app"
	}

	fmt.Fprintf(a.IO.Out, "\n📁 Setting up workdir: %s\n", workdir)
	if err := sshClient.Execute(ctx, fmt.Sprintf("mkdir -p %s", workdir), a.IO.Out, a.IO.Err); err != nil {
		return view, fmt.Errorf("failed to create workdir: %w", err)
	}

	// Step 4: Copy build context
	if buildContext != "" {
		fmt.Fprintf(a.IO.Out, "\n📦 Copying build context from: %s\n", buildContext)
		if err := copyBuildContext(ctx, sshClient, buildContext, workdir, df, a.IO.Out, a.IO.Err); err != nil {
			return view, fmt.Errorf("failed to copy build context: %w", err)
		}
	}

	// Step 5: Set environment variables
	if len(df.Env) > 0 {
		fmt.Fprintln(a.IO.Out, "\n🔧 Setting environment variables...")
		if err := setupEnvironment(ctx, sshClient, df.Env, req.BuildArgs, a.IO.Err); err != nil {
			return view, fmt.Errorf("failed to set environment: %w", err)
		}
	}

	// Step 6: Execute RUN commands
	runCommands := df.GetRunCommands()
	if len(runCommands) > 0 {
		fmt.Fprintf(a.IO.Out, "\n🔨 Executing %d build steps...\n", len(runCommands))
		for i, cmd := range runCommands {
			fmt.Fprintf(a.IO.Out, "   [%d/%d] %s\n", i+1, len(runCommands), truncateString(cmd, 60))
			fullCmd := fmt.Sprintf("cd %s && %s", workdir, cmd)
			if err := sshClient.Execute(ctx, fullCmd, a.IO.Out, a.IO.Err); err != nil {
				return view, fmt.Errorf("build step %d failed: %w", i+1, err)
			}
		}
	}
	view.TotalSteps = len(runCommands)

	// Step 7: Create commit (snapshot)
	fmt.Fprintln(a.IO.Out, "\n📸 Creating snapshot...")

	commitResp, err := a.Client.Vm.Commit(ctx, view.VMID, vers.VmCommitParams{})
	if err != nil {
		return view, fmt.Errorf("failed to create commit: %w", err)
	}

	view.CommitID = commitResp.CommitID
	fmt.Fprintf(a.IO.Out, "   Commit: %s\n", view.CommitID)

	// Save tag as alias for the commit if provided
	if req.Tag != "" {
		// Store tag -> commit mapping in aliases
		if err := utils.SetAlias(fmt.Sprintf("image:%s", req.Tag), view.CommitID); err != nil {
			fmt.Fprintf(a.IO.Err, "Warning: could not save image tag: %v\n", err)
		}
		view.Tag = req.Tag
	}

	// Clean up: delete the build VM (keep only the commit)
	fmt.Fprintln(a.IO.Out, "\n🧹 Cleaning up build VM...")
	if _, err := a.Client.Vm.Delete(ctx, view.VMID, vers.VmDeleteParams{}); err != nil {
		fmt.Fprintf(a.IO.Err, "Warning: could not delete build VM: %v\n", err)
	}

	return view, nil
}

// waitForSSH waits for SSH to become available
func waitForSSH(ctx context.Context, client *sshutil.Client, stdout io.Writer) error {
	maxAttempts := 60
	for i := 0; i < maxAttempts; i++ {
		err := client.Execute(ctx, "echo ready", io.Discard, io.Discard)
		if err == nil {
			return nil
		}
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(2 * time.Second):
			fmt.Fprint(stdout, ".")
		}
	}
	fmt.Fprintln(stdout)
	return fmt.Errorf("timeout waiting for VM SSH")
}

// copyBuildContext copies files based on COPY instructions
func copyBuildContext(ctx context.Context, client *sshutil.Client, buildContext, workdir string, df *docker.Dockerfile, stdout, stderr io.Writer) error {
	copies := df.GetCopyInstructions()

	if len(copies) == 0 {
		fmt.Fprintln(stdout, "   Copying entire build context...")
		return client.Upload(ctx, buildContext, workdir, true)
	}

	for _, copy := range copies {
		if len(copy.Args) < 2 {
			continue
		}
		src := copy.Args[0]
		dst := copy.Args[len(copy.Args)-1]

		srcPath := filepath.Join(buildContext, src)
		dstPath := dst
		if !filepath.IsAbs(dst) {
			dstPath = filepath.Join(workdir, dst)
		}

		fmt.Fprintf(stdout, "   COPY %s -> %s\n", src, dstPath)

		info, err := os.Stat(srcPath)
		if err != nil {
			// Try glob
			matches, _ := filepath.Glob(srcPath)
			if len(matches) == 0 {
				return fmt.Errorf("source not found: %s", src)
			}
			for _, m := range matches {
				mInfo, _ := os.Stat(m)
				if err := client.Upload(ctx, m, filepath.Join(dstPath, filepath.Base(m)), mInfo != nil && mInfo.IsDir()); err != nil {
					return err
				}
			}
			continue
		}

		if err := client.Upload(ctx, srcPath, dstPath, info.IsDir()); err != nil {
			return err
		}
	}

	return nil
}

// setupEnvironment sets environment variables in /etc/environment
func setupEnvironment(ctx context.Context, client *sshutil.Client, env map[string]string, additional []string, stderr io.Writer) error {
	var envLines []string

	for key, value := range env {
		envLines = append(envLines, fmt.Sprintf("%s=%q", key, value))
	}

	for _, envVar := range additional {
		envLines = append(envLines, envVar)
	}

	if len(envLines) == 0 {
		return nil
	}

	cmd := fmt.Sprintf("cat >> /etc/environment << 'EOF'\n%s\nEOF", strings.Join(envLines, "\n"))
	return client.Execute(ctx, cmd, io.Discard, stderr)
}

// truncateString truncates a string with ellipsis
func truncateString(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen-3] + "..."
}
