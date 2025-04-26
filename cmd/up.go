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
		// Get flag values for passing to subcommands
		memSize, _ := cmd.Flags().GetInt64("mem-size")
		vcpuCount, _ := cmd.Flags().GetInt64("vcpu-count")
		rootfs, _ := cmd.Flags().GetString("rootfs")
		kernel, _ := cmd.Flags().GetString("kernel")

		// First, run the build command
		fmt.Println("=== Building rootfs image ===")
		buildCmd := buildCmd

		// Pass along rootfs flag if set
		if rootfs != "" {
			if err := buildCmd.Flags().Set("rootfs", rootfs); err != nil {
				return fmt.Errorf("failed to set rootfs flag: %w", err)
			}
		}

		if err := buildCmd.RunE(buildCmd, nil); err != nil {
			return fmt.Errorf("build failed: %w", err)
		}

		// Then, run the run command
		fmt.Println("\n=== Starting development environment ===")
		runCmd := runCmd

		// Pass along all flags
		if memSize > 0 {
			if err := runCmd.Flags().Set("mem-size", fmt.Sprintf("%d", memSize)); err != nil {
				return fmt.Errorf("failed to set mem-size flag: %w", err)
			}
		}

		if vcpuCount > 0 {
			if err := runCmd.Flags().Set("vcpu-count", fmt.Sprintf("%d", vcpuCount)); err != nil {
				return fmt.Errorf("failed to set vcpu-count flag: %w", err)
			}
		}

		if rootfs != "" {
			if err := runCmd.Flags().Set("rootfs", rootfs); err != nil {
				return fmt.Errorf("failed to set rootfs flag: %w", err)
			}
		}

		if kernel != "" {
			if err := runCmd.Flags().Set("kernel", kernel); err != nil {
				return fmt.Errorf("failed to set kernel flag: %w", err)
			}
		}

		if err := runCmd.RunE(runCmd, args); err != nil {
			return fmt.Errorf("run failed: %w", err)
		}

		return nil
	},
}

func init() {
	rootCmd.AddCommand(upCmd)

	// Add flags to override toml configuration, mirroring those from run command
	upCmd.Flags().Int64("mem-size", 0, "Override memory size (MiB)")
	upCmd.Flags().Int64("vcpu-count", 0, "Override number of virtual CPUs")
	upCmd.Flags().String("rootfs", "", "Override rootfs name")
	upCmd.Flags().String("kernel", "", "Override kernel name")
}
