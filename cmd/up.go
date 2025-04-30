package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

// upCmd represents the up command
var upCmd = &cobra.Command{
	Use:   "up [cluster]",
	Short: "Build and start a development environment",
	Long:  `Build a rootfs image and start a Vers development environment according to the configuration in vers.toml.`,
	Args:  cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		// Load configuration just once
		config, err := loadConfig()
		if err != nil {
			return fmt.Errorf("failed to load configuration: %w", err)
		}

		// Apply flag overrides
		applyFlagOverrides(cmd, config)

		// Skip build step if rootfs is "default" or if builder is "none"
		if config.Rootfs.Name == "default" || config.Rootfs.Builder == "none" {
			fmt.Println("=== Building rootfs image ===")
			if err := BuildRootfs(config); err != nil {
				return fmt.Errorf("build failed: %w", err)
			}
		}

		// Then, run the environment
		fmt.Println("\n=== Starting development environment ===")
		if err := StartCluster(config, args); err != nil {
			return fmt.Errorf("run failed: %w", err)
		}

		return nil
	},
}

func init() {
	rootCmd.AddCommand(upCmd)

	// Add flags to override toml configuration, mirroring those from run command
	upCmd.Flags().Int64("mem-size", 512, "Override memory size (MiB)")
	upCmd.Flags().Int64("vcpu-count", 1, "Override number of virtual CPUs")
	upCmd.Flags().String("rootfs", "default", "Override rootfs name")
	upCmd.Flags().String("kernel", "default.bin", "Override kernel name")
}
