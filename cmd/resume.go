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

// resumeCmd represents the resume command
var resumeCmd = &cobra.Command{
	Use:   "resume [vm-id|alias]",
	Short: "Resume a paused VM",
	Long:  `Resume a paused Vers VM. If no VM ID or alias is provided, uses the current HEAD.`,
	Args:  cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		var vmID string
		var vmInfo *utils.VMInfo
		s := styles.NewKillStyles()

		// Initialize context
		baseCtx := context.Background()
		apiCtx, cancel := context.WithTimeout(baseCtx, 30*time.Second)
		defer cancel()

		// Determine VM ID to use - OPTIMIZED: minimal API calls
		if len(args) == 0 {
			// Use HEAD VM - get ID first (no API call)
			vmID, err := utils.GetCurrentHeadVM()
			if err != nil {
				return fmt.Errorf(s.NoData.Render("no VM ID provided and %w"), err)
			}
			fmt.Printf(s.HeadStatus.Render("Using current HEAD VM: "+vmID) + "\n")
		} else {
			// Use provided identifier - resolve it first (1 API call)
			vmInfo, err := utils.ResolveVMIdentifier(apiCtx, client, args[0])
			if err != nil {
				return fmt.Errorf(s.NoData.Render("failed to find VM: %w"), err)
			}
			vmID = vmInfo.ID
		}

		// Create resume request using SDK (always use the resolved ID)
		updateParams := vers.APIVmUpdateParams{
			VmPatchParams: vers.VmPatchParams{
				State: vers.F(vers.VmPatchParamsStateRunning),
			},
		}

		// Make API call to resume the VM - OPTIMIZED: get display info from response
		if vmInfo == nil {
			// We're resuming HEAD VM, show progress with ID first
			utils.ProgressCounter(1, 1, "Resuming VM", vmID, &s)
		} else {
			// We already have display name from resolution
			utils.ProgressCounter(1, 1, "Resuming VM", vmInfo.DisplayName, &s)
		}

		response, err := client.API.Vm.Update(apiCtx, vmID, updateParams)
		if err != nil {
			displayName := vmID
			if vmInfo != nil {
				displayName = vmInfo.DisplayName
			}
			return fmt.Errorf(s.NoData.Render("failed to resume VM '%s': %w"), displayName, err)
		}

		// Create VMInfo from response if we don't have it (HEAD case)
		if vmInfo == nil {
			vmInfo = utils.CreateVMInfoFromUpdateResponse(response.Data)
		}

		utils.SuccessMessage(fmt.Sprintf("VM '%s' resumed successfully", vmInfo.DisplayName), &s)
		fmt.Printf(s.HeadStatus.Render("VM state: %s\n"), response.Data.State)
		return nil
	},
}

func init() {
	rootCmd.AddCommand(resumeCmd)
}
