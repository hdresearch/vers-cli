package cmd

import (
	"context"
	"fmt"
	"time"

	"github.com/hdresearch/vers-cli/styles"
	vers "github.com/hdresearch/vers-sdk-go"
	"github.com/spf13/cobra"
)

// resumeCmd represents the resume command
var resumeCmd = &cobra.Command{
	Use:   "resume [vm-id]",
	Short: "Resume a paused VM",
	Long:  `Resume a paused Vers VM. If no VM ID is provided, uses the current HEAD.`,
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

		fmt.Printf(s.Progress.Render("Resuming VM '%s'...\n"), vmID)

		// Create resume request using SDK
		updateParams := vers.APIVmUpdateParams{
			UpdateVm: vers.UpdateVmParam{
				State: vers.F(vers.UpdateVmStateRunning),
			},
		}

		// Make API call to resume the VM
		response, err := client.API.Vm.Update(apiCtx, vmID, updateParams)
		if err != nil {
			return fmt.Errorf(s.NoData.Render("failed to resume VM '%s': %w"), vmID, err)
		}

		fmt.Printf(s.Success.Render("✓ VM '%s' resumed successfully\n"), response.Data.ID)
		fmt.Printf(s.HeadStatus.Render("VM state: %s\n"), response.Data.State)
		return nil
	},
}

func init() {
	rootCmd.AddCommand(resumeCmd)
}
