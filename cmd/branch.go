package cmd

import (
	"context"
	"fmt"
	"time"

	"github.com/hdresearch/vers-cli/internal/utils"
	"github.com/hdresearch/vers-cli/styles"
	"github.com/hdresearch/vers-sdk-go"
	"github.com/spf13/cobra"
)

var alias string

// branchCmd represents the branch command
var branchCmd = &cobra.Command{
	Use:   "branch [vm-id|alias]",
	Short: "Create a new VM from an existing VM",
	Long:  `Create a new VM (branch) from the state of an existing VM. If no VM ID or alias is provided, uses the current HEAD.`,
	Args:  cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		var vmID string
		var vmInfo *utils.VMInfo
		s := styles.NewBranchStyles()

		baseCtx := context.Background()
		apiCtx, cancel := context.WithTimeout(baseCtx, 30*time.Second)
		defer cancel()

		// Determine VM ID to use - OPTIMIZED: minimal API calls
		if len(args) == 0 {
			// Use HEAD VM - get ID first (no API call)
			headVMID, err := utils.GetCurrentHeadVM()
			if err != nil {
				return fmt.Errorf(s.Error.Render("no VM ID provided and %s"), err)
			}
			vmID = headVMID
			fmt.Printf(s.Tip.Render("Using current HEAD VM: ") + s.VMID.Render(vmID) + "\n")
		} else {
			// Use provided identifier - resolve it first (1 API call)
			resolvedVMInfo, err := utils.ResolveVMIdentifier(apiCtx, client, args[0])
			if err != nil {
				return fmt.Errorf(s.Error.Render("failed to find VM: %w"), err)
			}
			vmInfo = resolvedVMInfo
			vmID = vmInfo.ID
		}

		// Show progress with display name if we have it
		progressName := vmID
		if vmInfo != nil {
			progressName = vmInfo.DisplayName
		}
		fmt.Println(s.Progress.Render("Creating new VM from: " + progressName))

		body := vers.APIVmBranchParams{
			VmBranchParams: vers.VmBranchParams{},
		}
		if alias != "" {
			body.VmBranchParams.Alias = vers.F(alias)
		}

		// Make API call using the resolved VM ID
		response, err := client.API.Vm.Branch(apiCtx, vmID, body)
		if err != nil {
			return fmt.Errorf(s.Error.Render("failed to create branch from vm '%s': %w"), progressName, err)
		}
		branchInfo := response.Data

		// Success message
		fmt.Printf(s.Success.Render("✓ New VM created successfully!") + "\n")

		// VM details
		fmt.Printf(s.ListHeader.Render("New VM details:") + "\n")
		fmt.Printf(s.ListItem.Render(s.InfoLabel.Render("VM ID")+": "+s.VMID.Render(branchInfo.ID)) + "\n")

		if branchInfo.Alias != "" {
			fmt.Printf(s.ListItem.Render(s.InfoLabel.Render("Alias")+": "+s.BranchName.Render(branchInfo.Alias)) + "\n")
		}

		fmt.Printf(s.ListItem.Render(s.InfoLabel.Render("IP Address")+": "+s.CurrentState.Render(branchInfo.IPAddress)) + "\n")
		fmt.Printf(s.ListItem.Render(s.InfoLabel.Render("State")+": "+s.CurrentState.Render(string(branchInfo.State))) + "\n\n")

		// Check if user wants to switch to the new VM - OPTIMIZED: no extra API call
		if checkout, _ := cmd.Flags().GetBool("checkout"); checkout {
			// OPTIMIZED: Use branch response data directly, no extra API call needed!
			err := utils.SetHead(branchInfo.ID) // Direct HEAD update (no API call)
			if err != nil {
				warningMsg := fmt.Sprintf("WARNING: Failed to update HEAD: %v", err)
				fmt.Println(s.Warning.Render(warningMsg))
			} else {
				// Create display name from branch response
				displayName := branchInfo.Alias
				if displayName == "" {
					displayName = branchInfo.ID
				}
				fmt.Printf(s.Success.Render("✓ HEAD now points to: ") + s.BranchName.Render(displayName) + "\n")
			}
		} else {
			// Show tip about switching
			switchTarget := branchInfo.Alias
			if switchTarget == "" {
				switchTarget = branchInfo.ID
			}
			fmt.Printf(s.Tip.Render("Use --checkout or -c to switch to the new VM") + "\n")
			fmt.Printf(s.Tip.Render("Run 'vers checkout "+switchTarget+"' to switch to this VM") + "\n")
		}

		return nil
	},
}

func init() {
	rootCmd.AddCommand(branchCmd)

	// Define flags for the branch command
	branchCmd.Flags().StringVarP(&alias, "alias", "n", "", "Alias for the new VM")
	branchCmd.Flags().BoolP("checkout", "c", false, "Switch to the new VM after creation")
}
