package cmd

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"time"

	vers "github.com/hdresearch/vers-sdk-go"
	"github.com/spf13/cobra"
)

// runCmd represents the run command
var runCmd = &cobra.Command{
	Use:   "run [cluster]",
	Short: "Start a development environment",
	Long:  `Start a Vers development environment according to the configuration in vers.toml.`,
	Args:  cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		// Load configuration from vers.toml
		config, err := loadConfig()
		if err != nil {
			return fmt.Errorf("failed to load configuration: %w", err)
		}

		// Override with flags if provided
		applyFlagOverrides(cmd, config)

		return StartCluster(config, args)
	},
}

// StartCluster starts a development environment according to the provided configuration
func StartCluster(config *Config, args []string) error {
	baseCtx := context.Background()
	apiCtx, cancel := context.WithTimeout(baseCtx, 30*time.Second)
	defer cancel()

	// Create cluster parameters based on config
	clusterParams := vers.APIClusterNewParams{
		Create: vers.CreateParam{
			MemSizeMib: vers.F(config.Machine.MemSizeMib),
			VcpuCount:  vers.F(config.Machine.VcpuCount),
			RootfsName: vers.F(config.Rootfs.Name),
			KernelName: vers.F(config.Kernel.Name),
		},
	}

	fmt.Println("Sending request to start cluster...")
	response, err := client.API.Cluster.New(apiCtx, clusterParams)
	if err != nil {
		return err
	}
	clusterInfo := response.Data

	// Use information from the response
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
				return fmt.Errorf("Warning: Failed to update refs: %w\n", err)
			} else {
				fmt.Printf("Updated VM reference: %s -> %s\n", "refs/heads/main", vmID)
			}

			// HEAD already points to refs/heads/main from init, so we don't need to update it
			fmt.Println("HEAD is now pointing to the new VM")
		}
	}

	return nil
}

func init() {
	rootCmd.AddCommand(runCmd)

	// Add flags to override toml configuration
	runCmd.Flags().Int64("mem-size", 0, "Override memory size (MiB)")
	runCmd.Flags().Int64("vcpu-count", 0, "Override number of virtual CPUs")
	runCmd.Flags().String("rootfs", "", "Override rootfs name")
	runCmd.Flags().String("kernel", "", "Override kernel name")
}
