package cmd

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/hdresearch/vers-cli/internal/output"
	vers "github.com/hdresearch/vers-sdk-go"
	"github.com/spf13/cobra"
)

var (
	clusterAlias string
	vmAlias      string
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

	// Create base parameters
	params := vers.ClusterCreateRequestNewClusterParamsParamsParam{
		MemSizeMib:       vers.F(config.Machine.MemSizeMib),
		VcpuCount:        vers.F(config.Machine.VcpuCount),
		RootfsName:       vers.F(config.Rootfs.Name),
		KernelName:       vers.F(config.Kernel.Name),
		FsSizeClusterMib: vers.F(config.Machine.FsSizeClusterMib),
		FsSizeVmMib:      vers.F(config.Machine.FsSizeVmMib),
	}

	// Add aliases if provided
	if clusterAlias != "" {
		params.ClusterAlias = vers.F(clusterAlias)
	}
	if vmAlias != "" {
		params.VmAlias = vers.F(vmAlias)
	}

	// Create cluster parameters with modified params
	clusterParams := vers.APIClusterNewParams{
		ClusterCreateRequest: vers.ClusterCreateRequestNewClusterParamsParam{
			ClusterType: vers.F(vers.ClusterCreateRequestNewClusterParamsClusterTypeNew),
			Params:      vers.F(params),
		},
	}

	fmt.Println("Sending request to start cluster...")
	response, err := client.API.Cluster.New(apiCtx, clusterParams)
	if err != nil {
		return err
	}
	clusterInfo := response.Data

	// Build success and status output together
	result := output.New()
	result.WriteLinef("Cluster (ID: %s) started successfully with root vm '%s'.",
		clusterInfo.ID, clusterInfo.RootVmID)

	// Update HEAD to point to the new VM
	vmTarget := clusterInfo.RootVmID
	if vmAlias != "" {
		vmTarget = vmAlias // Use alias if provided
	}

	versDir := ".vers"
	if _, err := os.Stat(versDir); os.IsNotExist(err) {
		result.WriteLine("Warning: .vers directory not found. Run 'vers init' first.")
	} else {
		headFile := filepath.Join(versDir, "HEAD")
		if err := os.WriteFile(headFile, []byte(vmTarget+"\n"), 0644); err != nil {
			return fmt.Errorf("failed to update HEAD: %w", err)
		}
		result.WriteLinef("HEAD now points to: %s", vmTarget)
	}

	result.Print()
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
	runCmd.Flags().StringVarP(&clusterAlias, "cluster-alias", "n", "", "Set an alias for the cluster")
	runCmd.Flags().StringVarP(&vmAlias, "vm-alias", "N", "", "Set an alias for the root VM")
}
