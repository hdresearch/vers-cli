package cmd

import (
	"context"
	"fmt"

	"github.com/hdresearch/vers-cli/internal/handlers"
	pres "github.com/hdresearch/vers-cli/internal/presenters"
	"github.com/spf13/cobra"
)

var (
	dockerMemSize      int64
	dockerVcpuCount    int64
	dockerFsSize       int64
	dockerVMAlias      string
	dockerDetach       bool
	dockerPorts        []string
	dockerEnvVars      []string
	dockerInteractive  bool
	dockerBuildContext string
	dockerBuildTag     string
	dockerBuildNoCache bool
	dockerBuildArgs    []string
	dockerDockerfile   string
	composeProjectName string
	composeServices    []string
	composeNoDeps      bool
	composeFile        string
)

// dockerCmd represents the docker command group
var dockerCmd = &cobra.Command{
	Use:   "docker",
	Short: "Docker compatibility commands for Vers VMs",
	Long: `Run Docker-style commands that execute on Vers VMs instead of containers.

This provides a familiar Docker CLI experience while leveraging Vers VM capabilities:
  - Full VM isolation (not containers)
  - Persistent state with snapshots
  - Branch and restore
  - Full SSH access

Similar to how 'uv pip' provides pip compatibility or 'jj git' provides git compatibility,
'vers docker' provides Docker CLI compatibility on top of Vers VMs.

Examples:
  vers docker run ./Dockerfile            # Run Dockerfile in a Vers VM
  vers docker build -t myapp ./Dockerfile # Build and create a snapshot
  vers docker compose up                  # Start a compose project`,
}

// dockerRunCmd represents the docker run subcommand
var dockerRunCmd = &cobra.Command{
	Use:   "run [dockerfile]",
	Short: "Run a Dockerfile on a Vers VM",
	Long: `Parse a Dockerfile and execute it on a Vers VM.

This command:
  1. Creates a new Vers VM
  2. Parses the Dockerfile instructions
  3. Copies the build context to the VM
  4. Executes RUN commands
  5. Starts the application (CMD/ENTRYPOINT)

The VM persists after the command completes, allowing you to:
  - Connect via SSH (vers connect)
  - Create snapshots (vers commit)
  - Branch the state (vers branch)

Note: FROM instructions specify the expected base environment but Vers VMs
use their own base image. Common tools (Node.js, Python, etc.) may need
to be installed via RUN commands.

Examples:
  vers docker run                           # Use ./Dockerfile, current dir as context
  vers docker run ./Dockerfile              # Specify Dockerfile
  vers docker run -f ./Dockerfile.dev       # Use alternate Dockerfile
  vers docker run -c ./app ./Dockerfile     # Custom build context
  vers docker run -d ./Dockerfile           # Run detached
  vers docker run -N myapp ./Dockerfile     # Set VM alias
  vers docker run --mem 2048 ./Dockerfile   # Custom memory (2GB)`,
	Args: cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		dockerfilePath := dockerDockerfile
		if dockerfilePath == "" && len(args) > 0 {
			dockerfilePath = args[0]
		}

		// Use longer timeout for docker operations (same as build upload - 10 min)
		apiCtx, cancel := context.WithTimeout(context.Background(), application.Timeouts.BuildUpload)
		defer cancel()

		req := handlers.DockerRunReq{
			DockerfilePath: dockerfilePath,
			BuildContext:   dockerBuildContext,
			MemSizeMib:     dockerMemSize,
			VcpuCount:      dockerVcpuCount,
			FsSizeMib:      dockerFsSize,
			VMAlias:        dockerVMAlias,
			Detach:         dockerDetach,
			PortMappings:   dockerPorts,
			EnvVars:        dockerEnvVars,
			Interactive:    dockerInteractive,
		}

		view, err := handlers.HandleDockerRun(apiCtx, application, req)
		if err != nil {
			return err
		}

		pres.RenderDockerRun(application, view)
		return nil
	},
}

// dockerBuildCmd represents the docker build subcommand (creates a snapshot)
var dockerBuildCmd = &cobra.Command{
	Use:   "build [context]",
	Short: "Build a Dockerfile as a Vers snapshot",
	Long: `Parse a Dockerfile and create a Vers snapshot (commit) from it.

This is similar to 'docker build' - it creates a reusable image that can
be instantiated later. In Vers terms, this creates a committed VM snapshot.

The command:
  1. Creates a temporary build VM
  2. Copies the build context
  3. Executes all RUN instructions
  4. Creates a commit (snapshot)
  5. Deletes the build VM (only the snapshot remains)

Examples:
  vers docker build .                             # Build from current directory
  vers docker build -t myapp .                    # Tag the image
  vers docker build -f Dockerfile.prod .          # Use specific Dockerfile
  vers docker build --dockerfile ./Dockerfile .   # Explicit Dockerfile path`,
	Args: cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		// Build context is the argument (default: current dir)
		buildContext := "."
		if len(args) > 0 {
			buildContext = args[0]
		}

		// Use longer timeout for docker operations
		apiCtx, cancel := context.WithTimeout(context.Background(), application.Timeouts.BuildUpload)
		defer cancel()

		req := handlers.DockerBuildReq{
			DockerfilePath: dockerDockerfile,
			BuildContext:   buildContext,
			Tag:            dockerBuildTag,
			MemSizeMib:     dockerMemSize,
			VcpuCount:      dockerVcpuCount,
			FsSizeMib:      dockerFsSize,
			NoCache:        dockerBuildNoCache,
			BuildArgs:      dockerBuildArgs,
		}

		view, err := handlers.HandleDockerBuild(apiCtx, application, req)
		if err != nil {
			return err
		}

		pres.RenderDockerBuild(application, view)
		return nil
	},
}

// dockerComposeCmd represents the docker compose command group
var dockerComposeCmd = &cobra.Command{
	Use:   "compose",
	Short: "Docker Compose compatibility commands",
	Long: `Run Docker Compose-style commands that execute on Vers VMs.

Start multi-service applications defined in docker-compose.yml files.
Each service runs in its own Vers VM, providing full isolation.

Examples:
  vers docker compose up                    # Start all services
  vers docker compose up -d                 # Start detached
  vers docker compose up web api            # Start specific services`,
}

// dockerComposeUpCmd represents the docker compose up subcommand
var dockerComposeUpCmd = &cobra.Command{
	Use:   "up [services...]",
	Short: "Create and start services defined in docker-compose.yml",
	Long: `Create and start all services defined in docker-compose.yml.

This command:
  1. Parses the compose file
  2. Sorts services by dependency order
  3. Creates a Vers VM for each service
  4. Copies build contexts and runs setup commands
  5. Starts applications (if -d/--detach is specified)

Each service gets its own VM with an alias: <project>_<service>

Examples:
  vers docker compose up                    # Start all services
  vers docker compose up -d                 # Start in detached mode
  vers docker compose up web db             # Start only web and db
  vers docker compose up --no-deps web      # Start web without dependencies
  vers docker compose up -f compose.yml     # Use specific compose file`,
	RunE: func(cmd *cobra.Command, args []string) error {
		apiCtx, cancel := context.WithTimeout(context.Background(), application.Timeouts.BuildUpload)
		defer cancel()

		req := handlers.DockerComposeUpReq{
			ComposePath: composeFile,
			ProjectName: composeProjectName,
			Detach:      dockerDetach,
			Services:    args, // Services to start
			NoDeps:      composeNoDeps,
			EnvVars:     dockerEnvVars,
		}

		view, err := handlers.HandleDockerComposeUp(apiCtx, application, req)
		if err != nil {
			return err
		}

		pres.RenderDockerComposeUp(application, view)
		return nil
	},
}

// dockerComposePsCmd represents the docker compose ps subcommand
var dockerComposePsCmd = &cobra.Command{
	Use:   "ps",
	Short: "List running compose services",
	Long: `List all VMs that are part of a compose project.

Examples:
  vers docker compose ps
  vers docker compose ps -p myproject`,
	RunE: func(cmd *cobra.Command, args []string) error {
		// TODO: Implement by filtering VMs by project prefix alias
		return fmt.Errorf("docker compose ps is not yet implemented - use 'vers status' to list all VMs")
	},
}

// dockerComposeDownCmd represents the docker compose down subcommand
var dockerComposeDownCmd = &cobra.Command{
	Use:   "down",
	Short: "Stop and remove compose services",
	Long: `Stop and remove all VMs that are part of a compose project.

Examples:
  vers docker compose down
  vers docker compose down -p myproject`,
	RunE: func(cmd *cobra.Command, args []string) error {
		// TODO: Implement by filtering and killing VMs by project prefix
		return fmt.Errorf("docker compose down is not yet implemented - use 'vers kill <alias>' to stop individual services")
	},
}

func init() {
	rootCmd.AddCommand(dockerCmd)
	dockerCmd.AddCommand(dockerRunCmd)
	dockerCmd.AddCommand(dockerBuildCmd)
	dockerCmd.AddCommand(dockerComposeCmd)

	// Compose subcommands
	dockerComposeCmd.AddCommand(dockerComposeUpCmd)
	dockerComposeCmd.AddCommand(dockerComposePsCmd)
	dockerComposeCmd.AddCommand(dockerComposeDownCmd)

	// Common flags for docker run
	dockerRunCmd.Flags().StringVarP(&dockerBuildContext, "context", "c", "", "Build context directory (default: Dockerfile directory)")
	dockerRunCmd.Flags().StringVarP(&dockerDockerfile, "file", "f", "", "Path to Dockerfile (default: ./Dockerfile)")
	dockerRunCmd.Flags().Int64Var(&dockerMemSize, "mem", 0, "Memory size in MiB (default: 1024)")
	dockerRunCmd.Flags().Int64Var(&dockerVcpuCount, "vcpu", 0, "Number of vCPUs (default: 2)")
	dockerRunCmd.Flags().Int64Var(&dockerFsSize, "fs-size", 0, "Filesystem size in MiB (default: 4096)")
	dockerRunCmd.Flags().StringVarP(&dockerVMAlias, "name", "N", "", "Set an alias for the VM")
	dockerRunCmd.Flags().BoolVarP(&dockerDetach, "detach", "d", false, "Run in detached mode")
	dockerRunCmd.Flags().StringArrayVarP(&dockerPorts, "publish", "p", nil, "Publish port (host:container)")
	dockerRunCmd.Flags().StringArrayVarP(&dockerEnvVars, "env", "e", nil, "Set environment variables")
	dockerRunCmd.Flags().BoolVarP(&dockerInteractive, "interactive", "i", false, "Run interactively")

	// Flags for docker build
	dockerBuildCmd.Flags().StringVarP(&dockerBuildTag, "tag", "t", "", "Tag for the snapshot (e.g., myapp:latest)")
	dockerBuildCmd.Flags().StringVarP(&dockerDockerfile, "file", "f", "", "Path to Dockerfile (default: ./Dockerfile)")
	dockerBuildCmd.Flags().StringVar(&dockerDockerfile, "dockerfile", "", "Path to Dockerfile (alias for -f)")
	dockerBuildCmd.Flags().BoolVar(&dockerBuildNoCache, "no-cache", false, "Do not use cache (no-op, for compatibility)")
	dockerBuildCmd.Flags().StringArrayVar(&dockerBuildArgs, "build-arg", nil, "Set build-time variables")
	dockerBuildCmd.Flags().Int64Var(&dockerMemSize, "mem", 0, "Memory size in MiB (default: 1024)")
	dockerBuildCmd.Flags().Int64Var(&dockerVcpuCount, "vcpu", 0, "Number of vCPUs (default: 2)")
	dockerBuildCmd.Flags().Int64Var(&dockerFsSize, "fs-size", 0, "Filesystem size in MiB (default: 4096)")

	// Flags for docker compose up
	dockerComposeUpCmd.Flags().StringVarP(&composeFile, "file", "f", "", "Compose file path (default: docker-compose.yml)")
	dockerComposeUpCmd.Flags().StringVarP(&composeProjectName, "project-name", "p", "", "Project name (default: directory name)")
	dockerComposeUpCmd.Flags().BoolVarP(&dockerDetach, "detach", "d", false, "Run in detached mode")
	dockerComposeUpCmd.Flags().BoolVar(&composeNoDeps, "no-deps", false, "Don't start linked services")
	dockerComposeUpCmd.Flags().StringArrayVarP(&dockerEnvVars, "env", "e", nil, "Set environment variables")

	// Flags for docker compose ps/down
	dockerComposePsCmd.Flags().StringVarP(&composeProjectName, "project-name", "p", "", "Project name")
	dockerComposeDownCmd.Flags().StringVarP(&composeProjectName, "project-name", "p", "", "Project name")
}
