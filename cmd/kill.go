package cmd

import (
	"context"
	"fmt"
	"time"

	"github.com/hdresearch/vers-cli/internal/deletion"
	"github.com/hdresearch/vers-cli/styles"
	"github.com/spf13/cobra"
)

var (
	force     bool
	isCluster bool
	killAll   bool
)

var killCmd = &cobra.Command{
	Use:   "kill [vm-id|vm-alias|cluster-id|cluster-alias]...",
	Short: "Delete one or more VMs or clusters",
	Long: `Delete one or more VMs or clusters by ID or alias. Use -c flag for clusters, or -a flag to delete all clusters.
	
Examples:
  vers kill vm-123abc                    # Delete single VM by ID
  vers kill my-dev-vm my-test-vm         # Delete multiple VMs by alias
  vers kill -c cluster-456def            # Delete single cluster by ID
  vers kill -c my-cluster other-cluster  # Delete multiple clusters by alias
  vers kill -a                           # Delete ALL clusters (use with caution!)
  vers kill -a --force                   # Delete ALL clusters without confirmation`,
	Args: func(cmd *cobra.Command, args []string) error {
		if killAll {
			if len(args) > 0 {
				return fmt.Errorf("cannot specify target when using --all flag")
			}
			return nil
		}
		if len(args) == 0 {
			return fmt.Errorf("requires at least 1 arg(s), received 0")
		}
		return nil
	},
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
		defer cancel()

		s := styles.NewKillStyles()

		if killAll {
			processor := deletion.NewClusterProcessor(client, &s, ctx, force)
			return processor.DeleteAllClusters()
		}

		if isCluster {
			processor := deletion.NewClusterProcessor(client, &s, ctx, force)
			return processor.DeleteClusters(args)
		} else {
			processor := deletion.NewVMProcessor(client, &s, ctx, force)
			return processor.DeleteVMs(args)
		}
	},
}

func init() {
	rootCmd.AddCommand(killCmd)
	killCmd.Flags().BoolVarP(&force, "force", "f", false, "Force termination without confirmation")
	killCmd.Flags().BoolVarP(&isCluster, "cluster", "c", false, "Target is a cluster instead of a VM")
	killCmd.Flags().BoolVarP(&killAll, "all", "a", false, "Delete ALL clusters (use with extreme caution)")
}
