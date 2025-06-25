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
		var vmInfo *utils.VMInfo
		var err error
		s := styles.NewKillStyles()

		// Initialize context
		baseCtx := context.Background()
		apiCtx, cancel := context.WithTimeout(baseCtx, 30*time.Second)
		defer cancel()

		// Resolve VM identifier (HEAD, ID, or alias) to get VM info
		if len(args) == 0 {
			// Use HEAD VM
			vmInfo, err = utils.GetCurrentHeadVMInfo(apiCtx, client)
			if err != nil {
				return fmt.Errorf(s.NoData.Render("no VM ID provided and %w"), err)
			}
			fmt.Printf(s.Progress.Render("Using current HEAD VM: %s")+"\n", vmInfo.DisplayName)
		} else {
			// Use provided identifier (could be ID or alias)
			vmInfo, err = utils.ResolveVMIdentifier(apiCtx, client, args[0])
			if err != nil {
				return fmt.Errorf(s.NoData.Render("failed to find VM: %w"), err)
			}
		}

		// Use utils for progress message (show display name for UX)
		utils.ProgressCounter(1, 1, "Pausing VM", vmInfo.DisplayName, &s)

		// Create pause request using SDK (always use the resolved ID)
		updateParams := vers.APIVmUpdateParams{
			VmPatchParams: vers.VmPatchParams{
				State: vers.F(vers.VmPatchParamsStatePaused),
			},
		}

		// Make API call to pause the VM (use ID for API call)
		response, err := client.API.Vm.Update(apiCtx, vmInfo.ID, updateParams)
		if err != nil {
			return fmt.Errorf(s.NoData.Render("failed to pause VM '%s': %w"), vmInfo.DisplayName, err)
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
