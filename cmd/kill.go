package cmd

import (
	"context"
	"fmt"
	"time"

	"github.com/hdresearch/vers-cli/internal/deletion"
	"github.com/hdresearch/vers-cli/internal/utils"
	"github.com/hdresearch/vers-cli/styles"
	"github.com/spf13/cobra"
)

var (
	skipConfirmation bool
	recursive        bool
	isCluster        bool
	killAll          bool
)

var killCmd = &cobra.Command{
	Use:   "kill [vm-id|vm-alias|cluster-id|cluster-alias]...",
	Short: "Delete one or more VMs or clusters",
	Long: `Delete one or more VMs or clusters by ID or alias. Use -c flag for clusters, or -a flag to delete all clusters.
If no arguments are provided, deletes the current HEAD VM.
	
Examples:
  vers kill                              # Delete current HEAD VM
  vers kill vm-123abc                    # Delete single VM by ID
  vers kill my-dev-vm my-test-vm         # Delete multiple VMs by alias
  vers kill -c cluster-456def            # Delete single cluster by ID
  vers kill -c my-cluster other-cluster  # Delete multiple clusters by alias
  vers kill -a                           # Delete ALL clusters (use with caution!)
  vers kill -y                           # Delete HEAD VM without confirmation
  vers kill -r vm-with-children          # Recursively delete VM and all its children
  vers kill -y -r vm-with-children       # Skip confirmations AND delete children
  vers kill -a -y                        # Delete ALL clusters without confirmation`,
	Args: func(cmd *cobra.Command, args []string) error {
		if killAll {
			if len(args) > 0 {
				return fmt.Errorf("cannot specify target when using --all flag")
			}
			return nil
		}
		// Allow 0 or more arguments (0 means use HEAD)
		return nil
	},
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
		defer cancel()

		s := styles.NewKillStyles()

		if killAll {
			processor := deletion.NewClusterDeletionProcessor(client, &s, ctx, skipConfirmation, recursive)
			return processor.DeleteAllClusters()
		}

		// Handle the case where no arguments are provided
		if len(args) == 0 {
			// Use HEAD VM - optimized path since HEAD is always a VM ID
			headVMID, err := utils.GetCurrentHeadVM()
			if err != nil {
				return fmt.Errorf(s.NoData.Render("no arguments provided and %w"), err)
			}

			fmt.Printf(s.Progress.Render("Using current HEAD VM: %s")+"\n", headVMID)

			// Use optimized deletion path for HEAD
			processor := deletion.NewVMDeletionProcessor(client, &s, ctx, skipConfirmation, recursive)
			return processor.DeleteHeadVM(headVMID, headVMID)
		}

		// Delegate to appropriate processor
		if isCluster {
			processor := deletion.NewClusterDeletionProcessor(client, &s, ctx, skipConfirmation, recursive)
			return processor.DeleteMultipleClusters(args)
		} else {
			processor := deletion.NewVMDeletionProcessor(client, &s, ctx, skipConfirmation, recursive)
			return processor.DeleteMultipleVMs(args)
		}
	},
}

func init() {
	rootCmd.AddCommand(killCmd)
	killCmd.Flags().BoolVarP(&skipConfirmation, "yes", "y", false, "Skip confirmation prompts")
	killCmd.Flags().BoolVarP(&recursive, "recursive", "r", false, "Recursively delete all children")
	killCmd.Flags().BoolVarP(&isCluster, "cluster", "c", false, "Target is a cluster instead of a VM")
	killCmd.Flags().BoolVarP(&killAll, "all", "a", false, "Delete ALL clusters (use with extreme caution)")
}
