package cmd

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/google/uuid"
	vers "github.com/hdresearch/vers-sdk-go"
	"github.com/spf13/cobra"
)

// upCmd represents the up command
var upCmd = &cobra.Command{
	Use:   "up [cluster]",
	Short: "Start a development environment",
	Long:  `Start a Vers development environment according to the configuration in vers.toml.`,
	Args:  cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		clusterName := fmt.Sprintf("new-cluster-%s", uuid.New())
		if len(args) > 0 {
			clusterName = args[0]
		}

		fmt.Printf("Preparing cluster parameters for cluster: %s\n", clusterName)

		baseCtx := context.Background()
		client = vers.NewClient()

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
			return fmt.Errorf("failed to start cluster '%s': %w", clusterName, err)
		}
		// Use information from the response (adjust field names as needed)
		fmt.Printf("Cluster (ID: %s) started successfully with root vm '%s'.\n",
			clusterInfo.ID,
			clusterInfo.RootVmID,
		)

		// Store VM ID in version control system
		vmID := clusterInfo.RootVmID
		if vmID != "" {
			// Check if .vers directory exists
			versDir := ".vers"
			if _, err := os.Stat(versDir); os.IsNotExist(err) {
				fmt.Println("Warning: .vers directory not found. Run 'vers init' first.")
			} else {
				// Update refs/heads/main with VM ID
				mainRefPath := filepath.Join(versDir, "refs", "heads", "main")
				if err := os.WriteFile(mainRefPath, []byte(vmID+"\n"), 0644); err != nil {
					fmt.Printf("Warning: Failed to update refs: %v\n", err)
				} else {
					fmt.Printf("Updated VM reference: %s -> %s\n", "refs/heads/main", vmID)
				}

				// HEAD already points to refs/heads/main from init, so we don't need to update it
				fmt.Println("HEAD is now pointing to the new VM")
			}
		}

		return nil
	},
}

func init() {
	rootCmd.AddCommand(upCmd)

	// Add flags for all optional parameters
	upCmd.Flags().String("kernel", "", "Kernel name to use for the cluster")
	upCmd.Flags().Int64("mem", 0, "Memory size in MiB (0 = use default)")
	upCmd.Flags().String("rootfs", "", "Root filesystem name to use")
	upCmd.Flags().Int64("vcpu", 0, "Number of virtual CPUs (0 = use default)")
}
