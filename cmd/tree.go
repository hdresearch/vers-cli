package cmd

import (
	"context"
	"fmt"
	"time"

	"github.com/hdresearch/vers-cli/internal/presenters"
	svc "github.com/hdresearch/vers-cli/internal/services/tree"
	"github.com/hdresearch/vers-cli/internal/utils"
	"github.com/spf13/cobra"
)

var treeCmd = &cobra.Command{
	Use:   "tree [cluster-id|cluster-alias]",
	Short: "Print the tree of the cluster",
	Long:  `Print a visual tree representation of the cluster and its VMs. If no cluster ID or alias is provided, uses the cluster from current HEAD.`,
	Args:  cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		// Initialize context
		baseCtx := context.Background()
		apiCtx, cancel := context.WithTimeout(baseCtx, 30*time.Second)
		defer cancel()

		// Resolve cluster identifier
		if len(args) == 0 {
			// Get current VM ID from HEAD
			headVMID, err := utils.GetCurrentHeadVM()
			if err != nil {
				return fmt.Errorf("no cluster ID provided and %w", err)
			}

			fmt.Printf("Finding cluster for current HEAD VM: %s\n", headVMID)
			cluster, err := svc.GetClusterForHeadVM(apiCtx, client, headVMID)
			if err != nil {
				return fmt.Errorf("failed to resolve cluster for HEAD: %w", err)
			}
			return presenters.RenderTree(cluster, headVMID)

		} else {
			clusterIdentifier := args[0]

			// Fetch the cluster directly by ID or alias
			cluster, err := svc.GetClusterByIdentifier(apiCtx, client, clusterIdentifier)
			if err != nil {
				return fmt.Errorf("failed to get cluster '%s': %w", clusterIdentifier, err)
			}

			// Get HEAD VM for highlighting
			headVMID, err := utils.GetCurrentHeadVM()
			if err != nil {
				headVMID = ""
			}

			return presenters.RenderTree(cluster, headVMID)
		}
	},
}

func init() {
	rootCmd.AddCommand(treeCmd)
}
