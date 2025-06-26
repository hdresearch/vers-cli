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

// pauseCmd represents the pause command
var pauseCmd = &cobra.Command{
	Use:   "pause [vm-id|alias]",
	Short: "Pause a running VM",
	Long:  `Pause a running Vers VM. If no VM ID or alias is provided, uses the current HEAD.`,
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
			fmt.Printf(s.Progress.Render("Using current HEAD VM: %s")+"\n", vmID)
		} else {
			// Use provided identifier - resolve it first (1 API call)
			vmInfo, err := utils.ResolveVMIdentifier(apiCtx, client, args[0])
			if err != nil {
				return fmt.Errorf(s.NoData.Render("failed to find VM: %w"), err)
			}
			vmID = vmInfo.ID
		}

		// Create pause request using SDK (always use the resolved ID)
		updateParams := vers.APIVmUpdateParams{
			VmPatchParams: vers.VmPatchParams{
				State: vers.F(vers.VmPatchParamsStatePaused),
			},
		}

		// Make API call to pause the VM - OPTIMIZED: get display info from response
		if vmInfo == nil {
			// We're pausing HEAD VM, show progress with ID first
			utils.ProgressCounter(1, 1, "Pausing VM", vmID, &s)
		} else {
			// We already have display name from resolution
			utils.ProgressCounter(1, 1, "Pausing VM", vmInfo.DisplayName, &s)
		}

		response, err := client.API.Vm.Update(apiCtx, vmID, updateParams)
		if err != nil {
			displayName := vmID
			if vmInfo != nil {
				displayName = vmInfo.DisplayName
			}
			return fmt.Errorf(s.NoData.Render("failed to pause VM '%s': %w"), displayName, err)
		}

		// Create VMInfo from response if we don't have it (HEAD case)
		if vmInfo == nil {
			vmInfo = utils.CreateVMInfoFromUpdateResponse(response.Data)
		}

		// Use utils for success message (show display name for UX)
		successMsg := fmt.Sprintf("VM '%s' paused successfully", vmInfo.DisplayName)
		utils.SuccessMessage(successMsg, &s)

		fmt.Printf(s.HeadStatus.Render("VM state: %s\n"), response.Data.State)
		return nil
	},
}

func init() {
	rootCmd.AddCommand(pauseCmd)
}
