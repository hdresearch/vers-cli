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
		baseCtx := context.Background()

		apiCtx, cancel := context.WithTimeout(baseCtx, 30*time.Second)
		defer cancel()

		// Create new cluster params with the mandatory KernelName field
		clusterParams := vers.APIClusterNewParams{}

		// Handle optional parameters from flags
		memSize, _ := cmd.Flags().GetInt64("mem-size")
		if memSize > 0 {
			clusterParams.MemSizeMib = vers.F(memSize)
			fmt.Printf("Setting memory size to %d MiB\n", memSize)
		}

		rootfs, _ := cmd.Flags().GetString("rootfs")
		if rootfs != "" {
			clusterParams.RootfsName = vers.F(rootfs)
			fmt.Printf("Using rootfs: %s\n", rootfs)
		}

		vcpuCount, _ := cmd.Flags().GetInt64("vcpu")
		if vcpuCount > 0 {
			clusterParams.VcpuCount = vers.F(vcpuCount)
			fmt.Printf("Setting vCPU count to %d\n", vcpuCount)
		}

		kernelName, _ := cmd.Flags().GetString("kernel")
		if kernelName != "" {
			clusterParams.KernelName = vers.F(kernelName)
			fmt.Printf("Using kernel: %s\n", kernelName)
		}

		fmt.Println("Sending request to start cluster...")
		clusterInfo, err := client.API.Cluster.New(apiCtx, clusterParams)
		if err != nil {
			return fmt.Errorf("failed to load configuration: %w", err)
		}

		// Apply flag overrides
		applyFlagOverrides(cmd, config)

		// Skip build step if rootfs is "default" or if builder is "none"
		if config.Rootfs.Name != "default" && config.Builder.Name != "none" {
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
	upCmd.Flags().Int64("mem-size", 0, "Override memory size (MiB)")
	upCmd.Flags().Int64("vcpu-count", 0, "Override number of virtual CPUs")
	upCmd.Flags().String("rootfs", "", "Override rootfs name")
	upCmd.Flags().String("kernel", "", "Override kernel name")
}
