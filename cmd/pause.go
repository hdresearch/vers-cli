package cmd

import (
	"context"
	"fmt"
	"time"

	"github.com/hdresearch/vers-cli/styles"
	vers "github.com/hdresearch/vers-sdk-go"
	"github.com/spf13/cobra"
)

// pauseCmd represents the pause command
var pauseCmd = &cobra.Command{
	Use:   "pause [vm-id]",
	Short: "Pause a running VM",
	Long:  `Pause a running Vers VM. If no VM ID is provided, uses the current HEAD.`,
	Args:  cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		var vmID string
		s := styles.NewKillStyles()

		// If no VM ID provided, try to use the current HEAD
		if len(args) == 0 {
			var err error
			vmID, err = getCurrentHeadVM()
			if err != nil {
				return fmt.Errorf(s.NoData.Render("no VM ID provided and %w"), err)
			}
			fmt.Printf(s.HeadStatus.Render("Using current HEAD VM: "+vmID) + "\n")
		} else {
			vmID = args[0]
		}

		// Initialize context
		baseCtx := context.Background()
		apiCtx, cancel := context.WithTimeout(baseCtx, 30*time.Second)
		defer cancel()

		fmt.Printf(s.Progress.Render("Pausing VM '%s'...\n"), vmID)

		// Create pause request using SDK
		updateParams := vers.APIVmUpdateParams{
			VmPatchParams: vers.VmPatchParams{
				State: vers.F(vers.VmPatchParamsState("Paused")),
			},
		}

		// Make API call to pause the VM
		response, err := client.API.Vm.Update(apiCtx, vmID, updateParams)
		if err != nil {
			return fmt.Errorf(s.NoData.Render("failed to pause VM '%s': %w"), vmID, err)
		}

		fmt.Printf(s.Success.Render("✓ VM '%s' paused successfully\n"), response.Data.ID)
		fmt.Printf(s.HeadStatus.Render("VM state: %s\n"), response.Data.State)
		return nil
	},
}

func init() {
	rootCmd.AddCommand(pauseCmd)
}
