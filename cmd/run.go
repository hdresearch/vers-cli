package cmd

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
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
		Create: vers.CreateNewClusterParamsParam{
			ClusterType: vers.F(vers.CreateNewClusterParamsClusterTypeNew),
			Params: vers.F(vers.CreateNewClusterParamsParamsParam{
				MemSizeMib:       vers.F(config.Machine.MemSizeMib),
				VcpuCount:        vers.F(config.Machine.VcpuCount),
				RootfsName:       vers.F(config.Rootfs.Name),
				KernelName:       vers.F(config.Kernel.Name),
				FsSizeClusterMib: vers.F(config.Machine.FsSizeClusterMib),
				FsSizeVmMib:      vers.F(config.Machine.FsSizeVmMib),
			}),
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
			// Ensure refs/heads directory exists
			refsHeadsDir := filepath.Join(versDir, "refs", "heads")
			if err := os.MkdirAll(refsHeadsDir, 0755); err != nil {
				return fmt.Errorf("failed to create refs/heads directory: %w", err)
			}

			// Update refs/heads/main with VM ID
			mainRefPath := filepath.Join(versDir, "refs", "heads", "main")
			if err := os.WriteFile(mainRefPath, []byte(vmID+"\n"), 0644); err != nil {
				return fmt.Errorf("failed to update refs: %w", err)
			} else {
				fmt.Printf("Updated VM reference: %s -> %s\n", "refs/heads/main", vmID)
			}

			// Update HEAD to point to main branch (especially important if HEAD was detached)
			headFile := filepath.Join(versDir, "HEAD")
			headData, err := os.ReadFile(headFile)
			if err != nil {
				// HEAD file doesn't exist, create it pointing to main
				if err := os.WriteFile(headFile, []byte("ref: refs/heads/main\n"), 0644); err != nil {
					return fmt.Errorf("failed to create HEAD file: %w", err)
				}
				fmt.Println("HEAD is now pointing to the new VM")
			} else {
				headContent := string(headData)
				if strings.Contains(headContent, "DETACHED_HEAD") || !strings.Contains(headContent, "ref: refs/heads/main") {
					// HEAD is detached or pointing elsewhere, update it to main
					if err := os.WriteFile(headFile, []byte("ref: refs/heads/main\n"), 0644); err != nil {
						return fmt.Errorf("failed to update HEAD: %w", err)
					}
					fmt.Println("HEAD updated to point to main branch")
				} else {
					fmt.Println("HEAD is now pointing to the new VM")
				}
			}
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
	runCmd.Flags().Int64("fs-size-cluster", 0, "Override cluster filesystem size (MiB)")
	runCmd.Flags().Int64("fs-size-vm", 0, "Override VM filesystem size (MiB)")
	runCmd.Flags().Int64("size-cluster", 0, "Override total cluster size (MiB)")
}
