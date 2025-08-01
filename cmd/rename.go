package cmd

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/hdresearch/vers-cli/internal/output"
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
				return errors.New(s.NoData.Render("cluster ID or alias must be provided when renaming clusters"))
			} else {
				// Both old ID and new alias provided
				clusterInfo, err = utils.ResolveClusterIdentifier(apiCtx, client, args[0])
				if err != nil {
					return fmt.Errorf(s.NoData.Render("failed to find cluster: %w"), err)
				}
				newAlias = args[1]
			}

			// Build cluster rename output
			result := output.New()
			result.WriteStyledLinef(s.Progress, "Renaming cluster '%s' to '%s'...", clusterInfo.DisplayName, newAlias)

			// Create cluster rename request using the resolved cluster ID
			updateParams := vers.APIClusterUpdateParams{
				ClusterPatchRequest: vers.ClusterPatchRequestParam{
					Alias: vers.F(newAlias),
				},
			}

			// Make API call to rename the cluster
			response, err := client.API.Cluster.Update(apiCtx, clusterInfo.ID, updateParams)
			if err != nil {
				return fmt.Errorf(s.NoData.Render("failed to rename cluster '%s': %w"), clusterInfo.DisplayName, err)
			}

			result.WriteStyledLinef(s.Success, "✓ Cluster '%s' renamed to '%s'", response.Data.ID, response.Data.Alias).
				Print()
		} else {
			// Handle VM rename
			if len(args) == 1 {
				// Only alias provided, use HEAD for VM ID
				headVMID, err := utils.GetCurrentHeadVM()
				if err != nil {
					return fmt.Errorf(s.NoData.Render("no ID provided and %w"), err)
				}
				newAlias = args[0]

				// Build VM rename output for HEAD case
				result := output.New()
				result.WriteStyledLinef(s.Progress, "Using current HEAD VM: %s", headVMID).
					WriteStyledLinef(s.Progress, "Renaming VM '%s' to '%s'...", headVMID, newAlias)

				// Create VM rename request
				updateParams := vers.APIVmUpdateParams{
					VmPatchRequest: vers.VmPatchRequestParam{
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

				result.WriteStyledLinef(s.Success, "✓ VM '%s' renamed to '%s'", vmInfo.ID, response.Data.Alias).
					Print()
			} else {
				// Both ID and alias provided
				vmInfo, err := utils.ResolveVMIdentifier(apiCtx, client, args[0])
				if err != nil {
					return fmt.Errorf(s.NoData.Render("failed to find VM: %w"), err)
				}
				newAlias = args[1]

				// Build VM rename output for specified VM case
				result := output.New()
				result.WriteStyledLinef(s.Progress, "Renaming VM '%s' to '%s'...", vmInfo.DisplayName, newAlias)

				// Create VM rename request using the resolved VM ID
				updateParams := vers.APIVmUpdateParams{
					VmPatchRequest: vers.VmPatchRequestParam{
						Alias: vers.F(newAlias),
					},
				}

				// Make API call to rename the VM
				response, err := client.API.Vm.Update(apiCtx, vmInfo.ID, updateParams)
				if err != nil {
					return fmt.Errorf(s.NoData.Render("failed to rename VM '%s': %w"), vmInfo.DisplayName, err)
				}

				result.WriteStyledLinef(s.Success, "✓ VM '%s' renamed to '%s'", response.Data.ID, response.Data.Alias).
					Print()
			}
		}

		return nil
	},
}

func init() {
	rootCmd.AddCommand(renameCmd)
	renameCmd.Flags().BoolP("cluster", "c", false, "Rename a cluster instead of a VM")
}
