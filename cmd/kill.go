package cmd

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/hdresearch/vers-cli/styles"
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
		s := styles.NewKillStyles()

		// Initialize SDK client and context
		baseCtx := context.Background()
		apiCtx, cancel := context.WithTimeout(baseCtx, 30*time.Second)
		defer cancel()

		if isCluster {
			// Handle cluster deletion
			if !force {
				// Get cluster info to show what will be deleted
				response, err := client.API.Cluster.Get(apiCtx, targetID)
				if err != nil {
					return fmt.Errorf(s.Error.Render("failed to get cluster information: %w"), err)
				}
				cluster := response.Data

				// Show warning with cluster details
				fmt.Printf(s.Warning.Render("⚠ Warning: You are about to delete cluster '%s' containing %d VMs"),
					targetID, cluster.VmCount)
				fmt.Println()

				// Ask for confirmation
				fmt.Print("Are you sure you want to proceed? [y/N]: ")
				var input string
				fmt.Scanln(&input)

				if !strings.EqualFold(input, "y") && !strings.EqualFold(input, "yes") {
					fmt.Println(s.NoData.Render("Operation cancelled"))
					return nil
				}
			}

			fmt.Printf(s.Progress.Render("Deleting cluster '%s'...\n"), targetID)
			result, err := client.API.Cluster.Delete(apiCtx, targetID)
			if err != nil {
				return fmt.Errorf(s.Error.Render("failed to delete cluster: %w"), err)
			}

			// Print result
			if len(result.Data.Vms.Errors) > 0 || result.Data.FsError != "" {
				fmt.Println(s.Warning.Render("Some resources failed to delete:"))
				// Print FS error if exists
				if result.Data.FsError != "" {
					fmt.Printf(s.Warning.Render("  • %s\n"), result.Data.FsError)
				}
				// Print VM errors
				for _, error := range result.Data.Vms.Errors {
					fmt.Printf(s.Warning.Render("  • %s: %s\n"), error.ID, error.Error)
				}
			} else {
				fmt.Printf(s.Success.Render("✓ Cluster '%s' deleted successfully\n"), result.Data.ClusterID)
			}

		} else {
			// Handle VM deletion
			if force {
				fmt.Printf(s.Progress.Render("Force deleting VM '%s'...\n"), targetID)
			} else {
				fmt.Printf(s.Progress.Render("Deleting VM '%s'...\n"), targetID)
			}

			deleteParams := vers.APIVmDeleteParams{
				Recursive: vers.F(force),
			}
			result, err := client.API.Vm.Delete(apiCtx, targetID, deleteParams)
			if err != nil {
				return fmt.Errorf(s.Error.Render("failed to delete VM: %w"), err)
			}

			// Print result
			if len(result.Data.Errors) > 0 {
				fmt.Println(s.Warning.Render("One or more VMs failed to delete:"))
				for _, error := range result.Data.Errors {
					fmt.Printf(s.Warning.Render("  • %s: %s\n"), error.ID, error.Error)
				}
			} else {
				fmt.Printf(s.Success.Render("✓ VM '%s' deleted successfully\n"), targetID)
			}
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
