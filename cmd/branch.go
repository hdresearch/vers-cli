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
	Use:   "branch [vm-id]",
	Short: "Create a new VM from an existing VM",
	Long:  `Create a new VM (branch) from the state of an existing VM. If no VM ID is provided, uses the current HEAD.`,
	Args:  cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		var vmName string
		// Use KillStyles for consistency with other commands
		s := styles.NewKillStyles()
		// Keep BranchStyles for the specific formatting we need
		branchS := styles.NewBranchStyles()

		// If no VM ID provided, try to use the current HEAD
		if len(args) == 0 {
			var err error
			vmName, err = utils.GetCurrentHeadVM()
			if err != nil {
				return fmt.Errorf(s.Error.Render("no VM ID provided and %s"), err)
			}
			fmt.Printf(branchS.Tip.Render("Using current HEAD VM: ") + branchS.VMID.Render(vmName) + "\n")
		} else {
			vmName = args[0]
		}

		baseCtx := context.Background()
		apiCtx, cancel := context.WithTimeout(baseCtx, 30*time.Second)
		defer cancel()

		// Use utils for progress message
		utils.ProgressCounter(1, 1, "Creating new VM from", vmName, &s)

		body := vers.APIVmBranchParams{
			VmBranchParams: vers.VmBranchParams{},
		}
		if alias != "" {
			body.VmBranchParams.Alias = vers.F(alias)
		}

		response, err := client.API.Vm.Branch(apiCtx, vmName, body)
		if err != nil {
			return fmt.Errorf(s.Error.Render("failed to create branch from vm '%s': %w"), vmName, err)
		}
		branchInfo := response.Data

		// Use utils for success message
		utils.SuccessMessage("New VM created successfully!", &s)

		// VM details - keep the nice formatting from BranchStyles
		fmt.Printf(branchS.ListHeader.Render("New VM details:") + "\n")
		fmt.Printf(branchS.ListItem.Render(branchS.InfoLabel.Render("VM ID")+": "+branchS.VMID.Render(branchInfo.ID)) + "\n")

		if branchInfo.Alias != "" {
			fmt.Printf(branchS.ListItem.Render(branchS.InfoLabel.Render("Alias")+": "+branchS.BranchName.Render(branchInfo.Alias)) + "\n")
		}

		fmt.Printf(branchS.ListItem.Render(branchS.InfoLabel.Render("IP Address")+": "+branchS.CurrentState.Render(branchInfo.IPAddress)) + "\n")
		fmt.Printf(branchS.ListItem.Render(branchS.InfoLabel.Render("State")+": "+branchS.CurrentState.Render(string(branchInfo.State))) + "\n\n")

		// Check if user wants to switch to the new VM
		if checkout, _ := cmd.Flags().GetBool("checkout"); checkout {
			target := branchInfo.ID
			if branchInfo.Alias != "" {
				target = branchInfo.Alias
			}

			// Use utils for HEAD management
			if err := utils.SetHead(target); err != nil {
				utils.WarningMessage(fmt.Sprintf("Failed to update HEAD: %v", err), &s)
			} else {
				fmt.Printf(branchS.Success.Render("✓ HEAD now points to: ") + branchS.BranchName.Render(target) + "\n")
			}
		} else {
			// Show tip about switching
			switchTarget := branchInfo.Alias
			if switchTarget == "" {
				switchTarget = branchInfo.ID
			}
			fmt.Printf(branchS.Tip.Render("Use --checkout or -c to switch to the new VM") + "\n")
			fmt.Printf(branchS.Tip.Render("Run 'vers checkout "+switchTarget+"' to switch to this VM") + "\n")
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
