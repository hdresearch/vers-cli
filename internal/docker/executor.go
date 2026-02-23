package docker

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/hdresearch/vers-cli/internal/app"
	vmSvc "github.com/hdresearch/vers-cli/internal/services/vm"
	sshutil "github.com/hdresearch/vers-cli/internal/ssh"
	"github.com/hdresearch/vers-cli/internal/utils"
	vers "github.com/hdresearch/vers-sdk-go"
)

// RunConfig holds configuration for running a Dockerfile on a Vers VM
type RunConfig struct {
	DockerfilePath string
	BuildContext   string // Directory to use as build context
	MemSizeMib     int64
	VcpuCount      int64
	FsSizeMib      int64
	VMAlias        string
	Detach         bool     // Run in detached mode
	PortMappings   []string // Port mappings (host:container format)
	EnvVars        []string // Additional environment variables
	Interactive    bool     // Run interactively
}

// RunResult contains the result of running a Dockerfile
type RunResult struct {
	VMID          string
	VMAlias       string
	Dockerfile    *Dockerfile
	ExposedPorts  []string
	StartCommand  []string
	SetupComplete bool
	Running       bool
}

// Executor handles running Dockerfiles on Vers VMs
type Executor struct {
	app *app.App
}

// NewExecutor creates a new Dockerfile executor
func NewExecutor(app *app.App) *Executor {
	return &Executor{app: app}
}

// Run executes a Dockerfile by:
// 1. Creating a new Vers VM
// 2. Copying the build context
// 3. Running setup commands (RUN instructions)
// 4. Starting the application (CMD/ENTRYPOINT)
func (e *Executor) Run(ctx context.Context, cfg RunConfig, stdout, stderr io.Writer) (*RunResult, error) {
	result := &RunResult{}

	// Parse the Dockerfile
	df, err := ParseDockerfile(cfg.DockerfilePath)
	if err != nil {
		return nil, fmt.Errorf("failed to parse Dockerfile: %w", err)
	}
	result.Dockerfile = df
	result.ExposedPorts = df.ExposePorts
	result.StartCommand = df.GetStartCommand()

	fmt.Fprintln(stdout, "📄 Parsed Dockerfile")
	fmt.Fprintf(stdout, "   Base image: %s\n", df.BaseImage)
	if df.WorkDir != "" {
		fmt.Fprintf(stdout, "   Workdir: %s\n", df.WorkDir)
	}
	if len(df.ExposePorts) > 0 {
		fmt.Fprintf(stdout, "   Exposed ports: %s\n", strings.Join(df.ExposePorts, ", "))
	}

	// Step 1: Create a new VM
	fmt.Fprintln(stdout, "\n🚀 Creating Vers VM...")

	vmConfig := vers.NewRootRequestVmConfigParam{
		MemSizeMib: vers.F(cfg.MemSizeMib),
		VcpuCount:  vers.F(cfg.VcpuCount),
		FsSizeMib:  vers.F(cfg.FsSizeMib),
	}

	body := vers.VmNewRootParams{
		NewRootRequest: vers.NewRootRequestParam{
			VmConfig: vers.F(vmConfig),
		},
	}

	resp, err := e.app.Client.Vm.NewRoot(ctx, body)
	if err != nil {
		return nil, fmt.Errorf("failed to create VM: %w", err)
	}

	result.VMID = resp.VmID
	fmt.Fprintf(stdout, "   VM created: %s\n", result.VMID)

	// Set HEAD and alias
	if err := utils.SetHead(result.VMID); err != nil {
		fmt.Fprintf(stderr, "Warning: could not set HEAD: %v\n", err)
	}

	if cfg.VMAlias != "" {
		if err := utils.SetAlias(cfg.VMAlias, result.VMID); err != nil {
			fmt.Fprintf(stderr, "Warning: could not set alias: %v\n", err)
		}
		result.VMAlias = cfg.VMAlias
	}

	// Step 2: Get connection info and create SSH client
	info, err := vmSvc.GetConnectInfo(ctx, e.app.Client, result.VMID)
	if err != nil {
		return nil, fmt.Errorf("failed to get VM connection info: %w", err)
	}

	sshClient := sshutil.NewClient(info.Host, info.KeyPath, info.VMDomain)

	// Wait for VM to be ready
	fmt.Fprintln(stdout, "\n⏳ Waiting for VM to be ready...")
	if err := e.waitForSSH(ctx, sshClient, stdout); err != nil {
		return nil, fmt.Errorf("VM not ready: %w", err)
	}
	fmt.Fprintln(stdout, "   VM is ready!")

	// Step 3: Setup workdir
	workdir := df.WorkDir
	if workdir == "" {
		workdir = "/app"
	}
	fmt.Fprintf(stdout, "\n📁 Setting up workdir: %s\n", workdir)

	mkdirCmd := fmt.Sprintf("mkdir -p %s", workdir)
	if err := sshClient.Execute(ctx, mkdirCmd, stdout, stderr); err != nil {
		return nil, fmt.Errorf("failed to create workdir: %w", err)
	}

	// Step 4: Copy build context
	if cfg.BuildContext != "" {
		fmt.Fprintf(stdout, "\n📦 Copying build context from: %s\n", cfg.BuildContext)
		if err := e.copyBuildContext(ctx, sshClient, cfg.BuildContext, workdir, df, stdout, stderr); err != nil {
			return nil, fmt.Errorf("failed to copy build context: %w", err)
		}
	}

	// Step 5: Set environment variables
	if len(df.Env) > 0 || len(cfg.EnvVars) > 0 {
		fmt.Fprintln(stdout, "\n🔧 Setting environment variables...")
		if err := e.setupEnvironment(ctx, sshClient, df.Env, cfg.EnvVars, stderr); err != nil {
			return nil, fmt.Errorf("failed to set environment: %w", err)
		}
	}

	// Step 6: Run setup commands (RUN instructions)
	runCommands := df.GetRunCommands()
	if len(runCommands) > 0 {
		fmt.Fprintf(stdout, "\n🔨 Running %d setup commands...\n", len(runCommands))
		for i, cmd := range runCommands {
			fmt.Fprintf(stdout, "   [%d/%d] %s\n", i+1, len(runCommands), truncateString(cmd, 60))

			// Execute in workdir
			fullCmd := fmt.Sprintf("cd %s && %s", workdir, cmd)
			if err := sshClient.Execute(ctx, fullCmd, stdout, stderr); err != nil {
				return nil, fmt.Errorf("setup command failed: %s: %w", truncateString(cmd, 40), err)
			}
		}
	}
	result.SetupComplete = true

	// Step 7: Start the application
	startCmd := result.StartCommand
	if len(startCmd) > 0 {
		cmdStr := strings.Join(startCmd, " ")
		fullCmd := fmt.Sprintf("cd %s && %s", workdir, cmdStr)

		if cfg.Detach {
			fmt.Fprintf(stdout, "\n▶️  Starting application (detached): %s\n", cmdStr)
			// Run in background with nohup
			bgCmd := fmt.Sprintf("nohup sh -c '%s' > /tmp/app.log 2>&1 &", fullCmd)
			if err := sshClient.Execute(ctx, bgCmd, stdout, stderr); err != nil {
				return nil, fmt.Errorf("failed to start application: %w", err)
			}
			result.Running = true
			fmt.Fprintln(stdout, "   Application started in background")
			fmt.Fprintln(stdout, "   Logs: /tmp/app.log")
		} else if cfg.Interactive {
			fmt.Fprintf(stdout, "\n▶️  Starting application (interactive): %s\n", cmdStr)
			// Run interactively
			if err := sshClient.InteractiveCommand(ctx, fullCmd, e.app.IO.In, stdout, stderr); err != nil {
				// Don't return error for normal exit
				fmt.Fprintf(stderr, "Application exited: %v\n", err)
			}
		} else {
			fmt.Fprintf(stdout, "\n▶️  Starting application: %s\n", cmdStr)
			if err := sshClient.Execute(ctx, fullCmd, stdout, stderr); err != nil {
				return nil, fmt.Errorf("application failed: %w", err)
			}
		}
	} else {
		fmt.Fprintln(stdout, "\n✅ Setup complete (no CMD/ENTRYPOINT specified)")
		fmt.Fprintf(stdout, "   Connect with: vers connect %s\n", result.VMID)
	}

	return result, nil
}

// waitForSSH waits for SSH to become available on the VM
func (e *Executor) waitForSSH(ctx context.Context, client *sshutil.Client, stdout io.Writer) error {
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
			// Wait before retrying
			fmt.Fprint(stdout, ".")
		}
	}
	fmt.Fprintln(stdout)
	return fmt.Errorf("timeout waiting for VM SSH")
}

// copyBuildContext copies files based on COPY instructions
func (e *Executor) copyBuildContext(ctx context.Context, client *sshutil.Client, buildContext, workdir string, df *Dockerfile, stdout, stderr io.Writer) error {
	copies := df.GetCopyInstructions()

	if len(copies) == 0 {
		// No COPY instructions, copy everything
		fmt.Fprintln(stdout, "   Copying entire build context...")
		return e.copyDirectory(ctx, client, buildContext, workdir, stdout, stderr)
	}

	for _, copy := range copies {
		if len(copy.Args) < 2 {
			continue
		}

		src := copy.Args[0]
		dst := copy.Args[len(copy.Args)-1]

		// Resolve source path relative to build context
		srcPath := filepath.Join(buildContext, src)

		// Resolve destination path relative to workdir
		dstPath := dst
		if !filepath.IsAbs(dst) {
			dstPath = filepath.Join(workdir, dst)
		}

		fmt.Fprintf(stdout, "   COPY %s -> %s\n", src, dstPath)

		// Use SFTP to copy
		if err := e.copyPath(ctx, client, srcPath, dstPath, stdout, stderr); err != nil {
			return fmt.Errorf("failed to copy %s: %w", src, err)
		}
	}

	return nil
}

// copyDirectory copies an entire directory to the VM
func (e *Executor) copyDirectory(ctx context.Context, client *sshutil.Client, src, dst string, stdout, stderr io.Writer) error {
	return client.Upload(ctx, src, dst, true)
}

// copyPath copies a file or directory
func (e *Executor) copyPath(ctx context.Context, client *sshutil.Client, src, dst string, stdout, stderr io.Writer) error {
	// Check if source is a directory or file
	matches, err := filepath.Glob(src)
	if err != nil || len(matches) == 0 {
		// Try as exact path - check if it exists
		info, statErr := os.Stat(src)
		if statErr != nil {
			return fmt.Errorf("source not found: %s", src)
		}
		return client.Upload(ctx, src, dst, info.IsDir())
	}

	// Handle glob patterns
	for _, path := range matches {
		info, err := os.Stat(path)
		if err != nil {
			continue
		}

		dstPath := dst
		if len(matches) > 1 {
			// Multiple files, dst should be a directory
			dstPath = filepath.Join(dst, filepath.Base(path))
		}

		if err := client.Upload(ctx, path, dstPath, info.IsDir()); err != nil {
			return err
		}
	}

	return nil
}

// setupEnvironment sets environment variables in /etc/environment
func (e *Executor) setupEnvironment(ctx context.Context, client *sshutil.Client, env map[string]string, additional []string, stderr io.Writer) error {
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

	// Append to /etc/environment
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
