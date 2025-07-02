package cmd

import (
	"context"
	"fmt"
	"time"

	"github.com/hdresearch/vers-cli/internal/utils"
	"github.com/spf13/cobra"
)

var tag string

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
		if tag != "" {
			fmt.Printf("Tagging commit as: %s\n", tag)
		}

		// Get VM details for alias information
		fmt.Println("Creating commit...")
		if vmInfo == nil {
			vmResponse, err := client.API.Vm.Get(apiCtx, vmID)
			if err != nil {
				return fmt.Errorf("failed to get VM details: %w", err)
			}
			vmInfo = utils.CreateVMInfoFromGetResponse(vmResponse.Data)
		}

		// Call the SDK to commit the VM state
		response, err := client.API.Vm.Commit(apiCtx, vmInfo.ID)
		if err != nil {
			return fmt.Errorf("failed to commit VM '%s': %w", vmInfo.DisplayName, err)
		}
		commitResult := response.Data

		fmt.Printf("Successfully committed VM '%s'\n", vmInfo.DisplayName)
		fmt.Printf("Commit ID: %s\n", commitResult.ID)

		return nil
	},
}

func init() {
	rootCmd.AddCommand(commitCmd)

	// Define flags for the commit command
	commitCmd.Flags().StringVarP(&tag, "tag", "t", "", "Tag for this commit")
}
