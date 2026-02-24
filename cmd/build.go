package cmd

import (
	"context"
	"fmt"

	"github.com/hdresearch/vers-cli/internal/handlers"
	pres "github.com/hdresearch/vers-cli/internal/presenters"
	"github.com/hdresearch/vers-cli/internal/runconfig"
	"github.com/spf13/cobra"
)

var (
	buildDockerfile string
	buildTag        string
	buildContext    string
	buildNoCache    bool
	buildBuildArgs  []string
	buildMemSize    int64
	buildVcpuCount  int64
	buildFsSize     int64
)

// buildCmd represents the build command
var buildCmd = &cobra.Command{
	Use:   "build",
	Short: "Build a rootfs image",
	Long:  `Build a rootfs image according to the configuration in vers.toml and the Dockerfile in the current directory.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		// If --dockerfile is specified, delegate to docker build with deprecation warning
		if buildDockerfile != "" {
			fmt.Fprintln(cmd.ErrOrStderr(), "⚠️  DEPRECATED: 'vers build --dockerfile' is deprecated. Use 'vers docker build' instead.")
			fmt.Fprintln(cmd.ErrOrStderr())

			// Determine build context
			ctxDir := buildContext
			if ctxDir == "" {
				ctxDir = "."
			}

			apiCtx, cancel := context.WithTimeout(context.Background(), application.Timeouts.BuildUpload)
			defer cancel()

			req := handlers.DockerBuildReq{
				DockerfilePath: buildDockerfile,
				BuildContext:   ctxDir,
				Tag:            buildTag,
				MemSizeMib:     buildMemSize,
				VcpuCount:      buildVcpuCount,
				FsSizeMib:      buildFsSize,
				NoCache:        buildNoCache,
				BuildArgs:      buildBuildArgs,
			}

			view, err := handlers.HandleDockerBuild(apiCtx, application, req)
			if err != nil {
				return err
			}

			pres.RenderDockerBuild(application, view)
			return nil
		}

		// Original build behavior (rootfs from vers.toml)
		config, err := runconfig.Load()
		if err != nil {
			return fmt.Errorf("failed to load configuration: %w", err)
		}

		// Apply flag overrides
		applyFlagOverrides(cmd, config)

		apiCtx, cancel := context.WithTimeout(context.Background(), application.Timeouts.BuildUpload)
		defer cancel()
		view, err := handlers.HandleBuild(apiCtx, application, handlers.BuildReq{Config: config})
		if err != nil {
			return err
		}
		// Print a generic start line for parity
		fmt.Println("Creating tar archive of working directory...")
		fmt.Printf("Uploading rootfs archive as '%s'...\n", config.Rootfs.Name)
		pres.RenderBuild(application, view)
		return nil
	},
}

func init() {
	rootCmd.AddCommand(buildCmd)

	// Add flags to override toml configuration
	buildCmd.Flags().String("rootfs", "", "Override rootfs name")

	// Dockerfile build flags (deprecated - use 'vers docker build' instead)
	buildCmd.Flags().StringVar(&buildDockerfile, "dockerfile", "", "Path to Dockerfile (DEPRECATED: use 'vers docker build' instead)")
	buildCmd.Flags().StringVarP(&buildTag, "tag", "t", "", "Tag for the snapshot (e.g., myapp:latest)")
	buildCmd.Flags().StringVarP(&buildContext, "context", "c", "", "Build context directory (default: current directory)")
	buildCmd.Flags().BoolVar(&buildNoCache, "no-cache", false, "Do not use cache (no-op, for compatibility)")
	buildCmd.Flags().StringArrayVar(&buildBuildArgs, "build-arg", nil, "Set build-time variables")
	buildCmd.Flags().Int64Var(&buildMemSize, "mem", 0, "Memory size in MiB (default: 1024)")
	buildCmd.Flags().Int64Var(&buildVcpuCount, "vcpu", 0, "Number of vCPUs (default: 2)")
	buildCmd.Flags().Int64Var(&buildFsSize, "fs-size", 0, "Filesystem size in MiB (default: 4096)")

	// Mark dockerfile flag as deprecated
	buildCmd.Flags().MarkDeprecated("dockerfile", "use 'vers docker build -f <dockerfile>' instead")
}
