package cmd

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"time"

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
		s := styles.NewBranchStyles()

		// If no VM ID provided, try to use the current HEAD
		if len(args) == 0 {
			var err error
			vmName, err = getCurrentHeadVM()
			if err != nil {
				return fmt.Errorf(s.Error.Render("no VM ID provided and %s"), err)
			}
			fmt.Printf(s.Tip.Render("Using current HEAD VM: ") + s.VMID.Render(vmName) + "\n")
		} else {
			vmName = args[0]
		}

		baseCtx := context.Background()
		apiCtx, cancel := context.WithTimeout(baseCtx, 30*time.Second)
		defer cancel()

		fmt.Println(s.Progress.Render("Creating new VM from: " + vmName))

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

		// Check if user wants to switch to the new VM
		if checkout, _ := cmd.Flags().GetBool("checkout"); checkout {
			versDir := ".vers"
			headFile := filepath.Join(versDir, "HEAD")

			target := branchInfo.ID
			if branchInfo.Alias != "" {
				target = branchInfo.Alias
			}

			if err := os.WriteFile(headFile, []byte(target+"\n"), 0644); err != nil {
				fmt.Printf(s.Warning.Render("⚠ Warning: Failed to update HEAD: %v\n"), err)
			} else {
				fmt.Printf(s.Success.Render("✓ HEAD now points to: ") + s.BranchName.Render(target) + "\n")
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
	branchCmd.Flags().StringVarP(&alias, "alias", "a", "", "Alias for the new VM")
	branchCmd.Flags().BoolP("checkout", "c", false, "Switch to the new VM after creation")
}
