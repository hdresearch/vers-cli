package cmd

import (
	"context"
	"fmt"

	"github.com/hdresearch/vers-cli/internal/handlers"
	pres "github.com/hdresearch/vers-cli/internal/presenters"
	"github.com/spf13/cobra"
)

var (
	dockerMemSize    int64
	dockerVcpuCount  int64
	dockerFsSize     int64
	dockerVMAlias    string
	dockerDetach     bool
	dockerPorts      []string
	dockerEnvVars    []string
	dockerInteractive bool
	dockerBuildContext string
)

// dockerCmd represents the docker command group
var dockerCmd = &cobra.Command{
	Use:   "docker",
	Short: "Docker compatibility commands for Vers VMs",
	Long: `Run Docker-style commands that execute on Vers VMs instead of containers.

Vers VMs provide the same isolation as containers but with full VM capabilities:
  - Persistent state with snapshots
  - Branch and restore
  - Full SSH access
  - No container overhead

Examples:
  vers docker run ./Dockerfile        # Run Dockerfile in a Vers VM
  vers docker run -d ./Dockerfile     # Run detached
  vers docker run -f ./path/Dockerfile -c ./context/`,
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
  vers docker run                         # Use ./Dockerfile, current dir as context
  vers docker run ./Dockerfile            # Specify Dockerfile
  vers docker run -c ./app ./Dockerfile   # Custom build context
  vers docker run -d ./Dockerfile         # Run detached
  vers docker run -N myapp ./Dockerfile   # Set VM alias
  vers docker run --mem 2048 ./Dockerfile # Custom memory (2GB)`,
	Args: cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		dockerfilePath := ""
		if len(args) > 0 {
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
	Use:   "build [dockerfile]",
	Short: "Build a Dockerfile as a Vers snapshot",
	Long: `Parse a Dockerfile and create a Vers snapshot (commit) from it.

This is similar to 'docker build' - it creates a reusable image that can
be instantiated later. In Vers terms, this creates a committed VM snapshot.

Examples:
  vers docker build ./Dockerfile
  vers docker build -t myimage ./Dockerfile`,
	Args: cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		// TODO: Implement docker build as vers commit
		return fmt.Errorf("docker build is not yet implemented - use 'vers docker run' followed by 'vers commit'")
	},
}

func init() {
	rootCmd.AddCommand(dockerCmd)
	dockerCmd.AddCommand(dockerRunCmd)
	dockerCmd.AddCommand(dockerBuildCmd)

	// Flags for docker run
	dockerRunCmd.Flags().StringVarP(&dockerBuildContext, "context", "c", "", "Build context directory (default: Dockerfile directory)")
	dockerRunCmd.Flags().Int64Var(&dockerMemSize, "mem", 0, "Memory size in MiB (default: 1024)")
	dockerRunCmd.Flags().Int64Var(&dockerVcpuCount, "vcpu", 0, "Number of vCPUs (default: 2)")
	dockerRunCmd.Flags().Int64Var(&dockerFsSize, "fs-size", 0, "Filesystem size in MiB (default: 4096)")
	dockerRunCmd.Flags().StringVarP(&dockerVMAlias, "name", "N", "", "Set an alias for the VM")
	dockerRunCmd.Flags().BoolVarP(&dockerDetach, "detach", "d", false, "Run in detached mode")
	dockerRunCmd.Flags().StringArrayVarP(&dockerPorts, "publish", "p", nil, "Publish port (host:container)")
	dockerRunCmd.Flags().StringArrayVarP(&dockerEnvVars, "env", "e", nil, "Set environment variables")
	dockerRunCmd.Flags().BoolVarP(&dockerInteractive, "interactive", "i", false, "Run interactively")

	// Flags for docker build
	dockerBuildCmd.Flags().StringP("tag", "t", "", "Tag for the snapshot")
}
