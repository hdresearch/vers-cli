package cmd

import (
	"context"
	"fmt"
	"time"

	"github.com/hdresearch/vers-cli/internal/utils"
	"github.com/spf13/cobra"
)

// commitCmd represents the commit command
var commitCmd = &cobra.Command{
	Use:   "commit [vm-id|alias]",
	Short: "Commit the current state of the environment",
	Long:  `Save the current state of the Vers environment as a commit. If no VM ID or alias is provided, commits the current HEAD VM.`,
	Args:  cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		var vmID string
		var vmInfo *utils.VMInfo

		// Initialize the context and SDK client
		baseCtx := context.Background()
		apiCtx, cancel := context.WithTimeout(baseCtx, 60*time.Second)
		defer cancel()

		// Determine VM ID to use
		if len(args) > 0 {
			vmInfo, err := utils.ResolveVMIdentifier(apiCtx, client, args[0])
			if err != nil {
				return fmt.Errorf("failed to find VM: %w", err)
			}
			vmID = vmInfo.ID
			fmt.Printf("Using provided VM: %s\n", vmInfo.DisplayName)
		} else {
			// Use HEAD VM
			var err error
			vmID, err = utils.GetCurrentHeadVM()
			if err != nil {
				return fmt.Errorf("failed to get current VM: %w", err)
			}
			fmt.Printf("Using current HEAD VM: %s\n", vmID)
		}

		fmt.Printf("Creating commit for VM '%s'\n", vmID)

		// Get VM details for display
		fmt.Println("Creating commit...")
		if vmInfo == nil {
			vmInfo = &utils.VMInfo{
				ID:          vmID,
				DisplayName: vmID,
			}
		}

		response, err := client.Vm.Commit(apiCtx, vmInfo.ID)
		if err != nil {
			return fmt.Errorf("failed to commit VM '%s': %w", vmInfo.DisplayName, err)
		}

		fmt.Printf("Successfully committed VM '%s'\n", vmInfo.DisplayName)
		fmt.Printf("Commit ID: %s\n", response.CommitID)
		fmt.Printf("Host Architecture: %s\n", response.HostArchitecture)

		return nil
	},
}

func init() {
	rootCmd.AddCommand(commitCmd)
}
