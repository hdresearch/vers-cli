package cmd

import (
	"context"
	"fmt"
	"time"

	"github.com/hdresearch/vers-cli/styles"
	vers "github.com/hdresearch/vers-sdk-go"
	"github.com/spf13/cobra"
)

// renameCmd represents the rename command
var renameCmd = &cobra.Command{
	Use:   "rename [vm-id] [new-alias]",
	Short: "Rename a VM or cluster",
	Long:  `Rename a VM or cluster by setting a new alias. Use -c flag for clusters.`,
	Args:  cobra.ExactArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		id := args[0]
		newAlias := args[1]
		s := styles.NewKillStyles()

		// Initialize context
		baseCtx := context.Background()
		apiCtx, cancel := context.WithTimeout(baseCtx, 30*time.Second)
		defer cancel()

		// Check if this is a cluster rename
		isCluster, _ := cmd.Flags().GetBool("cluster")
		if isCluster {
			fmt.Printf(s.Progress.Render("Renaming cluster '%s' to '%s'...\n"), id, newAlias)

			// Create cluster rename request
			updateParams := vers.APIClusterUpdateParams{
				ClusterPatchParams: vers.ClusterPatchParams{
					Alias: vers.F(newAlias),
				},
			}

			// Make API call to rename the cluster
			response, err := client.API.Cluster.Update(apiCtx, id, updateParams)
			if err != nil {
				return fmt.Errorf(s.NoData.Render("failed to rename cluster '%s': %w"), id, err)
			}

			fmt.Printf(s.Success.Render("✓ Cluster '%s' renamed to '%s'\n"), response.Data.ID, response.Data.Alias)
		} else {
			fmt.Printf(s.Progress.Render("Renaming VM '%s' to '%s'...\n"), id, newAlias)

			// Create VM rename request
			updateParams := vers.APIVmUpdateParams{
				VmPatchParams: vers.VmPatchParams{
					Alias: vers.F(newAlias),
				},
			}

			// Make API call to rename the VM
			response, err := client.API.Vm.Update(apiCtx, id, updateParams)
			if err != nil {
				return fmt.Errorf(s.NoData.Render("failed to rename VM '%s': %w"), id, err)
			}

			fmt.Printf(s.Success.Render("✓ VM '%s' renamed to '%s'\n"), response.Data.ID, response.Data.Alias)
		}

		return nil
	},
}

func init() {
	rootCmd.AddCommand(renameCmd)
	renameCmd.Flags().BoolP("cluster", "c", false, "Rename a cluster instead of a VM")
}
