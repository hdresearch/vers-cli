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
func StartClusterFromCommit(config *Config, commitId string) error {
	baseCtx := context.Background()
	apiCtx, cancel := context.WithTimeout(baseCtx, 30*time.Second)
	defer cancel()

	// Create parameters for FromCluster variant
	params := vers.ClusterCreateRequestClusterFromCommitParamsParamsParam{
		// Older SDK uses CommitKey; newer may alias this. Prefer key-based field.
		CommitKey: vers.F(commitId),
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
		ClusterCreateRequest: vers.ClusterCreateRequestClusterFromCommitParamsParam{
			ClusterType: vers.F(vers.ClusterCreateRequestClusterFromCommitParamsClusterTypeFromCommit),
			Params:      vers.F(params),
		},
	}

	fmt.Printf("Sending request to start cluster from commit %s...\n", commitId)
	response, err := client.API.Cluster.New(apiCtx, clusterParams)
	if err != nil {
		return err
	}
	clusterInfo := response.Data

	// Use information from the response
	fmt.Printf("Cluster (ID: %s) started successfully from commit %s with root vm '%s'.\n",
		clusterInfo.ID,
		commitId,
		clusterInfo.RootVmID,
	)

	// Update HEAD to point to the new VM (simplified architecture)
	vmTarget := clusterInfo.RootVmID
	if commitVmAlias != "" {
		vmTarget = commitVmAlias // Use alias if provided
	}

	versDir := ".vers"
	if _, err := os.Stat(versDir); os.IsNotExist(err) {
		fmt.Println("Warning: .vers directory not found. Run 'vers init' first.")
	} else {
		headFile := filepath.Join(versDir, "HEAD")
		if err := os.WriteFile(headFile, []byte(vmTarget+"\n"), 0644); err != nil {
			return fmt.Errorf("failed to update HEAD: %w", err)
		}
		fmt.Printf("HEAD now points to: %s (from commit %s)\n", vmTarget, commitId)
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
