package cmd

import (
	"context"
	"fmt"

	"github.com/hdresearch/vers-cli/internal/handlers"
	pres "github.com/hdresearch/vers-cli/internal/presenters"
	"github.com/hdresearch/vers-cli/internal/runconfig"
	"github.com/spf13/cobra"
)

// buildCmd represents the build command
var buildCmd = &cobra.Command{
	Use:   "build",
	Short: "Build a rootfs image",
	Long:  `Build a rootfs image according to the configuration in vers.toml and the Dockerfile in the current directory.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		// Load configuration from vers.toml
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
	buildCmd.Flags().String("dockerfile", "", "Dockerfile path")
}
