package cmd

import (
	"context"
	"fmt"
	"strings"
	"time"

	vers "github.com/hdresearch/vers-sdk-go"
	"github.com/spf13/cobra"
)

var force bool
var isCluster bool

// killCmd represents the kill command
var killCmd = &cobra.Command{
	Use:   "kill [-c] <vm-id|cluster-id>",
	Short: "Forcefully terminate a VM or cluster",
	Long:  `Forcefully terminate a VM or cluster in the Vers environment. Use -c flag to specify a cluster.`,
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		targetID := args[0]
		s := NewKillStyles()

		// Initialize SDK client and context
		baseCtx := context.Background()
		apiCtx, cancel := context.WithTimeout(baseCtx, 30*time.Second)
		defer cancel()

		if isCluster {
			// Handle cluster deletion
			if !force {
				// Get cluster info to show what will be deleted
				cluster, err := client.API.Cluster.Get(apiCtx, targetID)
				if err != nil {
					return fmt.Errorf(s.Error.Render("failed to get cluster information: %v"), err)
				}

				// Show warning with cluster details
				fmt.Printf(s.Warning.Render("⚠ Warning: You are about to delete cluster '%s' containing %d VMs\n"), 
					targetID, cluster.VmCount)
				
				// Ask for confirmation
				fmt.Print("Are you sure you want to proceed? [y/N]: ")
				var response string
				fmt.Scanln(&response)
				
				if !strings.EqualFold(response, "y") && !strings.EqualFold(response, "yes") {
					fmt.Println(s.NoData.Render("Operation cancelled"))
					return nil
				}
			}

			fmt.Printf(s.Progress.Render("Deleting cluster '%s'...\n"), targetID)
			_, err := client.API.Cluster.Delete(apiCtx, targetID)
			if err != nil {
				return fmt.Errorf(s.Error.Render("failed to delete cluster: %v"), err)
			}
			fmt.Printf(s.Success.Render("✓ Cluster '%s' deleted successfully\n"), targetID)

		} else {
			// Handle VM deletion
			if force {
				fmt.Printf(s.Progress.Render("Force deleting VM '%s'...\n"), targetID)
			} else {
				fmt.Printf(s.Progress.Render("Deleting VM '%s'...\n"), targetID)
			}

			deleteParams := vers.APIVmDeleteParams{
				Recursive: vers.F(false),
			}
			err := client.API.Vm.Delete(apiCtx, targetID, deleteParams)
			if err != nil {
				return fmt.Errorf(s.Error.Render("failed to delete VM: %v"), err)
			}
			fmt.Printf(s.Success.Render("✓ VM '%s' deleted successfully\n"), targetID)
		}

		return nil
	},
}

func init() {
	rootCmd.AddCommand(killCmd)

	// Define flags for the kill command
	killCmd.Flags().BoolVarP(&force, "force", "f", false, "Force termination without confirmation")
	killCmd.Flags().BoolVarP(&isCluster, "cluster", "c", false, "Target is a cluster instead of a VM")
} 