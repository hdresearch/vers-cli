package cmd

import (
	"context"
	"fmt"
	"time"

	"github.com/hdresearch/vers-cli/internal/utils"
	"github.com/hdresearch/vers-cli/styles"
	vers "github.com/hdresearch/vers-sdk-go"
	"github.com/spf13/cobra"
)

// renameCmd represents the rename command
var renameCmd = &cobra.Command{
	Use:   "rename [vm-id|alias|cluster-id|cluster-alias] [new-alias]",
	Short: "Rename a VM or cluster",
	Long:  `Rename a VM or cluster by setting a new alias. Use -c flag for clusters. If no ID is provided, uses the current HEAD VM.`,
	Args: func(cmd *cobra.Command, args []string) error {
		// Allow 1 or 2 arguments
		if len(args) < 1 || len(args) > 2 {
			return fmt.Errorf("accepts 1 or 2 arg(s), received %d", len(args))
		}
		return nil
	},
	RunE: func(cmd *cobra.Command, args []string) error {
		var newAlias string
		s := styles.NewKillStyles()

		// Initialize context
		baseCtx := context.Background()
		apiCtx, cancel := context.WithTimeout(baseCtx, 30*time.Second)
		defer cancel()

		// Check if this is a cluster rename
		isCluster, _ := cmd.Flags().GetBool("cluster")

		if isCluster {
			// Handle cluster rename
			var clusterInfo *utils.ClusterInfo
			var err error

			if len(args) == 1 {
				// Only one argument provided, this doesn't make sense for clusters since we can't use HEAD
				return fmt.Errorf(s.NoData.Render("cluster ID or alias must be provided when renaming clusters"))
			} else {
				// Both old ID and new alias provided
				clusterInfo, err = utils.ResolveClusterIdentifier(apiCtx, client, args[0])
				if err != nil {
					return fmt.Errorf(s.NoData.Render("failed to find cluster: %w"), err)
				}
				newAlias = args[1]
			}

			fmt.Printf(s.Progress.Render("Renaming cluster '%s' to '%s'...\n"), clusterInfo.DisplayName, newAlias)

			// Create cluster rename request using the resolved cluster ID
			updateParams := vers.APIClusterUpdateParams{
				ClusterPatchParams: vers.ClusterPatchParams{
					Alias: vers.F(newAlias),
				},
			}

			// Make API call to rename the cluster
			response, err := client.API.Cluster.Update(apiCtx, clusterInfo.ID, updateParams)
			if err != nil {
				return fmt.Errorf(s.NoData.Render("failed to rename cluster '%s': %w"), clusterInfo.DisplayName, err)
			}

			fmt.Printf(s.Success.Render("✓ Cluster '%s' renamed to '%s'\n"), response.Data.ID, response.Data.Alias)
		} else {
			// Handle VM rename
			if len(args) == 1 {
				// Only alias provided, use HEAD for VM ID
				headVMID, err := utils.GetCurrentHeadVM()
				if err != nil {
					return fmt.Errorf(s.NoData.Render("no ID provided and %w"), err)
				}
				fmt.Printf(s.Progress.Render("Using current HEAD VM: %s")+"\n", headVMID)
				newAlias = args[0]

				fmt.Printf(s.Progress.Render("Renaming VM '%s' to '%s'...\n"), headVMID, newAlias)

				// Create VM rename request
				updateParams := vers.APIVmUpdateParams{
					VmPatchParams: vers.VmPatchParams{
						Alias: vers.F(newAlias),
					},
				}

				// Make API call to rename the VM
				response, err := client.API.Vm.Update(apiCtx, headVMID, updateParams)
				if err != nil {
					return fmt.Errorf(s.NoData.Render("failed to rename VM '%s': %w"), headVMID, err)
				}

				// Create VMInfo from response
				vmInfo := utils.CreateVMInfoFromUpdateResponse(response.Data)

				fmt.Printf(s.Success.Render("✓ VM '%s' renamed to '%s'\n"), vmInfo.ID, response.Data.Alias)
			} else {
				// Both ID and alias provided
				vmInfo, err := utils.ResolveVMIdentifier(apiCtx, client, args[0])
				if err != nil {
					return fmt.Errorf(s.NoData.Render("failed to find VM: %w"), err)
				}
				newAlias = args[1]

				fmt.Printf(s.Progress.Render("Renaming VM '%s' to '%s'...\n"), vmInfo.DisplayName, newAlias)

				// Create VM rename request using the resolved VM ID
				updateParams := vers.APIVmUpdateParams{
					VmPatchParams: vers.VmPatchParams{
						Alias: vers.F(newAlias),
					},
				}

				// Make API call to rename the VM
				response, err := client.API.Vm.Update(apiCtx, vmInfo.ID, updateParams)
				if err != nil {
					return fmt.Errorf(s.NoData.Render("failed to rename VM '%s': %w"), vmInfo.DisplayName, err)
				}

				fmt.Printf(s.Success.Render("✓ VM '%s' renamed to '%s'\n"), response.Data.ID, response.Data.Alias)
			}
		}

		return nil
	},
}

func init() {
	rootCmd.AddCommand(renameCmd)
	renameCmd.Flags().BoolP("cluster", "c", false, "Rename a cluster instead of a VM")
}
