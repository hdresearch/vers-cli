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
		s := styles.NewKillStyles() // Use unified styles

		// If no VM ID provided, try to use the current HEAD
		if len(args) == 0 {
			var err error
			vmName, err = utils.GetCurrentHeadVM()
			if err != nil {
				return fmt.Errorf(s.Error.Render("no VM ID provided and %s"), err)
			}
			fmt.Printf(s.Warning.Render("Using current HEAD VM: ") + s.Success.Render(vmName) + "\n")
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

		// VM details using unified styles
		fmt.Printf(s.Progress.Render("New VM details:") + "\n")
		fmt.Printf("  VM ID: %s\n", s.Success.Render(branchInfo.ID))

		if branchInfo.Alias != "" {
			fmt.Printf("  Alias: %s\n", s.Success.Render(branchInfo.Alias))
		}

		fmt.Printf("  IP Address: %s\n", s.Success.Render(branchInfo.IPAddress))
		fmt.Printf("  State: %s\n\n", s.Success.Render(string(branchInfo.State)))

		// Check if user wants to switch to the new VM
		if checkout, _ := cmd.Flags().GetBool("checkout"); checkout {
			target := branchInfo.ID
			if branchInfo.Alias != "" {
				target = branchInfo.Alias
			}

			// Use utils for HEAD management
			if err := utils.SetHead(target); err != nil {
				warningMsg := fmt.Sprintf("WARNING: Failed to update HEAD: %v", err)
				fmt.Println(s.Warning.Render(warningMsg))
			} else {
				fmt.Printf(s.Success.Render("✓ HEAD now points to: ") + s.Success.Render(target) + "\n")
			}
		} else {
			// Show tip about switching
			switchTarget := branchInfo.Alias
			if switchTarget == "" {
				switchTarget = branchInfo.ID
			}
			fmt.Printf(s.Warning.Render("Use --checkout or -c to switch to the new VM") + "\n")
			fmt.Printf(s.Warning.Render("Run 'vers checkout "+switchTarget+"' to switch to this VM") + "\n")
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
