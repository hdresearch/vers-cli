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

var (
	commitClusterAlias string
	commitVmAlias      string
)

// runCommitCmd represents the run-commit command
var runCommitCmd = &cobra.Command{
	Use:   "run-commit [commit-key]",
	Short: "Start a development environment from a commit",
	Long:  `Start a Vers development environment from an existing commit using its commit key.`,
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		commitKey := args[0]
		if commitKey == "" {
			return fmt.Errorf("commit key is required")
		}

		// Load configuration from vers.toml for any overrides
		config, err := loadConfig()
		if err != nil {
			return fmt.Errorf("failed to load configuration: %w", err)
		}

		// Override with flags if provided
		applyFlagOverrides(cmd, config)

		return StartClusterFromCommit(config, commitKey)
	},
}

// StartClusterFromCommit starts a development environment from an existing commit
func StartClusterFromCommit(config *Config, commitKey string) error {
	baseCtx := context.Background()
	apiCtx, cancel := context.WithTimeout(baseCtx, 30*time.Second)
	defer cancel()

	// Create parameters for FromCluster variant
	params := vers.ClusterCreateParamsClusterFromCommitParamsParams{
		CommitKey: vers.F(commitKey),
	}

	// Add aliases if provided
	if commitClusterAlias != "" {
		params.ClusterAlias = vers.F(commitClusterAlias)
	}
	if commitVmAlias != "" {
		params.VmAlias = vers.F(commitVmAlias)
	}

	// Apply any configuration overrides
	if config.Machine.FsSizeClusterMib > 0 {
		params.FsSizeClusterMib = vers.F(config.Machine.FsSizeClusterMib)
	}

	// Create cluster parameters
	clusterParams := vers.APIClusterNewParams{
		ClusterCreateParams: vers.ClusterCreateParamsClusterFromCommitParams{
			ClusterType: vers.F(vers.ClusterCreateParamsClusterFromCommitParamsClusterTypeFromCommit),
			Params:      vers.F(params),
		},
	}

	fmt.Printf("Sending request to start cluster from commit %s...\n", commitKey)
	response, err := client.API.Cluster.New(apiCtx, clusterParams)
	if err != nil {
		return err
	}
	clusterInfo := response.Data

	// Use information from the response
	fmt.Printf("Cluster (ID: %s) started successfully from commit %s with root vm '%s'.\n",
		clusterInfo.ID,
		commitKey,
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
			fmt.Println("HEAD is now pointing to the VM from commit", commitKey)
		}
	}

	return nil
}

func init() {
	rootCmd.AddCommand(runCommitCmd)

	// Add flags to override configuration (same as run command)
	runCommitCmd.Flags().Int64("fs-size-cluster", 0, "Override cluster filesystem size (MiB)")
	runCommitCmd.Flags().StringVarP(&commitClusterAlias, "cluster-alias", "n", "", "Set an alias for the cluster")
	runCommitCmd.Flags().StringVarP(&commitVmAlias, "vm-alias", "N", "", "Set an alias for the root VM")
}
