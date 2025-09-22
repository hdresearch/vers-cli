package cmd

import (
	"context"
	"fmt"
	"time"

	"github.com/hdresearch/vers-cli/internal/handlers"
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
		// -a and -c are mutually exclusive
		if killAll && isCluster {
			return fmt.Errorf("cannot use --all and --cluster together")
		}

		// -a requires no arguments
		if killAll && len(args) > 0 {
			return fmt.Errorf("cannot specify targets when using --all")
		}

		// -c requires at least one argument
		if isCluster && !killAll && len(args) == 0 {
			return fmt.Errorf("--cluster requires at least one cluster identifier")
		}

		// -r only makes sense for VMs, not clusters
		if recursive && isCluster {
			return fmt.Errorf("--recursive only applies to VMs, not clusters")
		}

		// Allow 0 or more arguments for VM operations (0 means use HEAD)
		return nil
	},
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
		defer cancel()
		req := handlers.KillReq{
			Targets:          args,
			SkipConfirmation: skipConfirmation,
			Recursive:        recursive,
			IsCluster:        isCluster,
			KillAll:          killAll,
		}
		return handlers.HandleKill(ctx, application, req)
	},
}

func init() {
	rootCmd.AddCommand(killCmd)
	killCmd.Flags().BoolVarP(&skipConfirmation, "yes", "y", false, "Skip confirmation prompts")
	killCmd.Flags().BoolVarP(&recursive, "recursive", "r", false, "Recursively delete all children")
	killCmd.Flags().BoolVarP(&isCluster, "cluster", "c", false, "Target is a cluster instead of a VM")
	killCmd.Flags().BoolVarP(&killAll, "all", "a", false, "Delete ALL clusters (use with extreme caution)")
}
