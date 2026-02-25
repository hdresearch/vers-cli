package docker

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/hdresearch/vers-cli/internal/app"
	vmSvc "github.com/hdresearch/vers-cli/internal/services/vm"
	sshutil "github.com/hdresearch/vers-cli/internal/ssh"
	"github.com/hdresearch/vers-cli/internal/utils"
	vers "github.com/hdresearch/vers-sdk-go"
)

// ComposeConfig holds configuration for running a compose file
type ComposeConfig struct {
	ComposePath string   // Path to docker-compose.yml
	ProjectName string   // Project name (default: directory name)
	Detach      bool     // Run in detached mode
	Services    []string // Specific services to run (empty = all)
	NoDeps      bool     // Don't start dependencies
	EnvVars     []string // Additional environment variables
}

// ServiceResult contains the result of starting a single service
type ServiceResult struct {
	Name          string
	VMID          string
	VMAlias       string
	Ports         []string
	Running       bool
	SetupComplete bool
	Error         error
}

// ComposeResult contains the result of running docker compose up
type ComposeResult struct {
	ProjectName   string
	Services      []ServiceResult
	TotalServices int
}

// ComposeExecutor handles running docker-compose files on Vers VMs
type ComposeExecutor struct {
	app *app.App
}

// NewComposeExecutor creates a new compose executor
func NewComposeExecutor(app *app.App) *ComposeExecutor {
	return &ComposeExecutor{app: app}
}

// Up starts all services in the compose file
func (e *ComposeExecutor) Up(ctx context.Context, cfg ComposeConfig, stdout, stderr io.Writer) (*ComposeResult, error) {
	result := &ComposeResult{
		ProjectName: cfg.ProjectName,
	}

	// Parse the compose file
	compose, err := ParseComposeFile(cfg.ComposePath)
	if err != nil {
		return nil, fmt.Errorf("failed to parse compose file: %w", err)
	}

	// Get base directory for resolving relative paths
	baseDir := filepath.Dir(cfg.ComposePath)
	if baseDir == "" {
		baseDir = "."
	}

	// Parse and sort services by dependency order
	services, err := compose.ParseServices(baseDir)
	if err != nil {
		return nil, err
	}

	// Filter services if specific ones were requested
	if len(cfg.Services) > 0 && !cfg.NoDeps {
		services = filterServicesWithDeps(services, cfg.Services)
	} else if len(cfg.Services) > 0 {
		services = filterServices(services, cfg.Services)
	}

	result.TotalServices = len(services)
	fmt.Fprintf(stdout, "🐳 Starting %d services from %s\n", len(services), cfg.ComposePath)
	fmt.Fprintln(stdout)

	// Track VMs by service name for inter-service communication
	vmMap := &sync.Map{}

	// Start services in dependency order
	for _, svc := range services {
		fmt.Fprintf(stdout, "━━━ Starting service: %s ━━━\n", svc.Name)

		svcResult, err := e.startService(ctx, cfg, svc, vmMap, stdout, stderr)
		if err != nil {
			svcResult = ServiceResult{
				Name:  svc.Name,
				Error: err,
			}
			fmt.Fprintf(stderr, "   ❌ Failed: %v\n", err)
		}

		result.Services = append(result.Services, svcResult)

		if svcResult.VMID != "" {
			vmMap.Store(svc.Name, svcResult.VMID)
		}

		fmt.Fprintln(stdout)
	}

	return result, nil
}

// startService starts a single service
func (e *ComposeExecutor) startService(ctx context.Context, cfg ComposeConfig, svc ParsedService, vmMap *sync.Map, stdout, stderr io.Writer) (ServiceResult, error) {
	result := ServiceResult{
		Name:  svc.Name,
		Ports: svc.Ports,
	}

	// Create VM alias from project name and service name
	vmAlias := fmt.Sprintf("%s_%s", cfg.ProjectName, svc.Name)
	result.VMAlias = vmAlias

	// Check if this service has a Dockerfile or is image-based
	var df *Dockerfile
	if svc.Dockerfile != "" {
		var err error
		df, err = ParseDockerfile(svc.Dockerfile)
		if err != nil {
			return result, fmt.Errorf("failed to parse Dockerfile: %w", err)
		}
		fmt.Fprintf(stdout, "   📄 Using Dockerfile: %s\n", svc.Dockerfile)
	} else if svc.Image != "" {
		fmt.Fprintf(stdout, "   📦 Image-based service: %s\n", svc.Image)
		fmt.Fprintf(stdout, "   ⚠️  Note: Using Vers base image (not %s)\n", svc.Image)
		// Create a minimal Dockerfile equivalent
		df = &Dockerfile{
			BaseImage: svc.Image,
			Env:       make(map[string]string),
		}
	} else {
		return result, fmt.Errorf("service has no build or image specified")
	}

	// Create VM
	fmt.Fprintln(stdout, "   🚀 Creating Vers VM...")

	vmConfig := vers.NewRootRequestVmConfigParam{
		MemSizeMib: vers.F(svc.MemSizeMib),
		VcpuCount:  vers.F(svc.VcpuCount),
		FsSizeMib:  vers.F(int64(4096)), // Default 4GB
	}

	body := vers.VmNewRootParams{
		NewRootRequest: vers.NewRootRequestParam{
			VmConfig: vers.F(vmConfig),
		},
	}

	resp, err := e.app.Client.Vm.NewRoot(ctx, body)
	if err != nil {
		return result, fmt.Errorf("failed to create VM: %w", err)
	}

	result.VMID = resp.VmID
	fmt.Fprintf(stdout, "   VM: %s\n", result.VMID)

	// Set alias
	if err := utils.SetAlias(vmAlias, result.VMID); err != nil {
		fmt.Fprintf(stderr, "   Warning: could not set alias: %v\n", err)
	}

	// Get connection info
	info, err := vmSvc.GetConnectInfo(ctx, e.app.Client, result.VMID)
	if err != nil {
		return result, fmt.Errorf("failed to get VM connection info: %w", err)
	}

	sshClient := sshutil.NewClient(info.Host, info.KeyPath, info.VMDomain)

	// Wait for VM to be ready
	fmt.Fprintln(stdout, "   ⏳ Waiting for VM...")
	if err := e.waitForSSH(ctx, sshClient); err != nil {
		return result, fmt.Errorf("VM not ready: %w", err)
	}

	// Setup workdir
	workdir := svc.WorkingDir
	if workdir == "" && df.WorkDir != "" {
		workdir = df.WorkDir
	}
	if workdir == "" {
		workdir = "/app"
	}

	mkdirCmd := fmt.Sprintf("mkdir -p %s", workdir)
	if err := sshClient.Execute(ctx, mkdirCmd, io.Discard, stderr); err != nil {
		return result, fmt.Errorf("failed to create workdir: %w", err)
	}

	// Copy build context if available
	if svc.BuildContext != "" {
		fmt.Fprintf(stdout, "   📦 Copying build context...\n")
		if err := e.copyBuildContext(ctx, sshClient, svc.BuildContext, workdir, df, stderr); err != nil {
			return result, fmt.Errorf("failed to copy build context: %w", err)
		}
	}

	// Merge environment variables
	env := make(map[string]string)
	for k, v := range df.Env {
		env[k] = v
	}
	for k, v := range svc.Environment {
		env[k] = v
	}
	// Add service discovery env vars (other services' VMs)
	vmMap.Range(func(key, value interface{}) bool {
		svcName := key.(string)
		vmID := value.(string)
		// Services can reach each other via their aliases
		env[fmt.Sprintf("%s_HOST", strings.ToUpper(svcName))] = vmAlias
		env[fmt.Sprintf("%s_VM", strings.ToUpper(svcName))] = vmID
		return true
	})

	// Set environment variables
	if len(env) > 0 {
		if err := e.setupEnvironment(ctx, sshClient, env, cfg.EnvVars, stderr); err != nil {
			return result, fmt.Errorf("failed to set environment: %w", err)
		}
	}

	// Run setup commands
	runCommands := df.GetRunCommands()
	if len(runCommands) > 0 {
		fmt.Fprintf(stdout, "   🔨 Running %d setup commands...\n", len(runCommands))
		for _, cmd := range runCommands {
			fullCmd := fmt.Sprintf("cd %s && %s", workdir, cmd)
			if err := sshClient.Execute(ctx, fullCmd, stdout, stderr); err != nil {
				return result, fmt.Errorf("setup command failed: %w", err)
			}
		}
	}
	result.SetupComplete = true

	// Determine start command
	startCmd := svc.Command
	if len(startCmd) == 0 {
		startCmd = df.GetStartCommand()
	}

	// Start the application
	if len(startCmd) > 0 {
		cmdStr := strings.Join(startCmd, " ")
		fullCmd := fmt.Sprintf("cd %s && %s", workdir, cmdStr)

		if cfg.Detach {
			fmt.Fprintf(stdout, "   ▶️  Starting: %s\n", truncateString(cmdStr, 50))
			bgCmd := fmt.Sprintf("nohup sh -c '%s' > /tmp/app.log 2>&1 &", fullCmd)
			if err := sshClient.Execute(ctx, bgCmd, io.Discard, stderr); err != nil {
				return result, fmt.Errorf("failed to start application: %w", err)
			}
			result.Running = true
		} else {
			fmt.Fprintf(stdout, "   ℹ️  Ready to start: %s\n", truncateString(cmdStr, 50))
		}
	}

	fmt.Fprintf(stdout, "   ✅ Service %s ready\n", svc.Name)
	return result, nil
}

// waitForSSH waits for SSH to become available
func (e *ComposeExecutor) waitForSSH(ctx context.Context, client *sshutil.Client) error {
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
		}
	}
	return fmt.Errorf("timeout waiting for VM SSH")
}

// copyBuildContext copies the build context to the VM
func (e *ComposeExecutor) copyBuildContext(ctx context.Context, client *sshutil.Client, buildContext, workdir string, df *Dockerfile, stderr io.Writer) error {
	copies := df.GetCopyInstructions()

	if len(copies) == 0 {
		// Copy everything
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

		info, err := os.Stat(srcPath)
		if err != nil {
			// Try glob
			matches, _ := filepath.Glob(srcPath)
			if len(matches) == 0 {
				continue
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

// setupEnvironment sets environment variables
func (e *ComposeExecutor) setupEnvironment(ctx context.Context, client *sshutil.Client, env map[string]string, additional []string, stderr io.Writer) error {
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

// filterServicesWithDeps filters to requested services plus their dependencies
func filterServicesWithDeps(services []ParsedService, requested []string) []ParsedService {
	requestedSet := make(map[string]bool)
	for _, name := range requested {
		requestedSet[name] = true
	}

	// Add all dependencies recursively
	svcMap := make(map[string]ParsedService)
	for _, svc := range services {
		svcMap[svc.Name] = svc
	}

	var addDeps func(name string)
	addDeps = func(name string) {
		if svc, ok := svcMap[name]; ok {
			requestedSet[name] = true
			for _, dep := range svc.DependsOn {
				addDeps(dep)
			}
		}
	}

	for _, name := range requested {
		addDeps(name)
	}

	// Filter
	var result []ParsedService
	for _, svc := range services {
		if requestedSet[svc.Name] {
			result = append(result, svc)
		}
	}
	return result
}

// filterServices filters to only the requested services
func filterServices(services []ParsedService, requested []string) []ParsedService {
	requestedSet := make(map[string]bool)
	for _, name := range requested {
		requestedSet[name] = true
	}

	var result []ParsedService
	for _, svc := range services {
		if requestedSet[svc.Name] {
			result = append(result, svc)
		}
	}
	return result
}
