package cmd

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/hdresearch/vers-cli/styles"
	"github.com/spf13/cobra"
)

var fromBranch string
var branchName string

// branchCmd represents the branch command
var branchCmd = &cobra.Command{
	Use:   "branch [vm-id]",
	Short: "Branch a machine",
	Long:  `Branch the state of a given machine. If no VM ID is provided, uses the current HEAD.`,
	Args:  cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		var vmName string
		s := styles.NewBranchStyles()

		// If no VM ID provided, try to use the current HEAD
		if len(args) == 0 {
			// Read the HEAD file to find the current branch
			versDir := ".vers"
			headFile := filepath.Join(versDir, "HEAD")

			// Check if .vers directory and HEAD file exist
			if _, err := os.Stat(headFile); os.IsNotExist(err) {
				return fmt.Errorf(s.Error.Render("no VM ID provided and HEAD not found. Run 'vers init' first"))
			}

			// Read HEAD file
			headData, err := os.ReadFile(headFile)
			if err != nil {
				return fmt.Errorf(s.Error.Render("error reading HEAD: %v"), err)
			}

			// Parse the HEAD content
			headContent := string(bytes.TrimSpace(headData))
			var refPath string

			// Check if HEAD is a symbolic ref or direct ref
			if strings.HasPrefix(headContent, "ref: ") {
				// It's a symbolic ref, extract the path
				refPath = strings.TrimPrefix(headContent, "ref: ")

				// Read the actual reference file
				refFile := filepath.Join(versDir, refPath)
				refData, err := os.ReadFile(refFile)
				if err != nil {
					return fmt.Errorf(s.Error.Render("error reading reference '%s': %v"), refPath, err)
				}

				// Get the VM ID from the reference file
				vmName = string(bytes.TrimSpace(refData))
			} else {
				// HEAD directly contains a VM ID
				vmName = headContent
			}

			if vmName == "" {
				return fmt.Errorf(s.Error.Render("could not determine current VM ID from HEAD"))
			}

			fmt.Printf(s.Tip.Render("Using current HEAD VM: ") + s.VMID.Render(vmName) + "\n")
		} else {
			vmName = args[0]
		}

		// We'll set the final branch name after we have the new VM ID
		// Temporary placeholder for branch name display
		tempBranchName := branchName
		if tempBranchName == "" {
			tempBranchName = "[auto-generated]"
		}

		baseCtx := context.Background()
		apiCtx, cancel := context.WithTimeout(baseCtx, 30*time.Second)
		defer cancel()

		fmt.Println(s.Progress.Render("Creating branch..."))
		response, err := client.API.Vm.Branch(apiCtx, vmName)
		if err != nil {
			return fmt.Errorf(s.Error.Render("failed to create branch of vm '%s': %v"), vmName, err)
		}
		branchInfo := response.Data

		// Store the branch VM ID in version control system
		branchVmID := branchInfo.ID
		if branchVmID != "" {
			// If no explicit branch name provided, use the new branch VM ID
			if branchName == "" {
				// Use the full VM ID as branch name
				branchName = branchVmID
			}

			// Check if .vers directory exists
			versDir := ".vers"
			if _, err := os.Stat(versDir); os.IsNotExist(err) {
				fmt.Println(s.Warning.Render("⚠ Warning: .vers directory not found. Run 'vers init' first."))
			} else {
				// Sanitize branch name (remove invalid characters for filenames)
				safeBranchName := strings.ReplaceAll(branchName, "/", "-")
				safeBranchName = strings.ReplaceAll(safeBranchName, "\\", "-")

				// Create branch ref file
				branchRefPath := filepath.Join(versDir, "refs", "heads", safeBranchName)
				if err := os.WriteFile(branchRefPath, []byte(branchVmID+"\n"), 0644); err != nil {
					fmt.Printf(s.Warning.Render("⚠ Warning: Failed to create branch ref: %v\n"), err)
				}

				// Branch creation success
				fmt.Printf(s.Success.Render("✓ Branch created successfully!") + "\n")

				// Optionally, switch HEAD to the new branch
				if checkout, _ := cmd.Flags().GetBool("checkout"); checkout {
					headFile := filepath.Join(versDir, "HEAD")
					newRef := fmt.Sprintf("ref: refs/heads/%s\n", safeBranchName)
					if err := os.WriteFile(headFile, []byte(newRef), 0644); err != nil {
						fmt.Printf(s.Warning.Render("⚠ Warning: Failed to update HEAD: %v\n"), err)
					} else {
						fmt.Printf(s.Success.Render("✓ HEAD now points to: ") + s.BranchName.Render("refs/heads/"+safeBranchName) + "\n")
					}
				} else {
					// Show message indicating HEAD was not changed
					currentBranch := getCurrentBranchName(versDir)
					fmt.Printf(s.HeadStatus.Render("HEAD: On branch '"+currentBranch+"'") + "\n")
				}
			}
		}

		// Branch details
		fmt.Printf(s.ListHeader.Render("Branch details:") + "\n")
		fmt.Printf(s.ListItem.Render(s.InfoLabel.Render("New VM ID")+": "+s.VMID.Render(branchInfo.ID)) + "\n")
		fmt.Printf(s.ListItem.Render(s.InfoLabel.Render("IP Address")+": "+s.CurrentState.Render(branchInfo.IPAddress)) + "\n")
		fmt.Printf(s.ListItem.Render(s.InfoLabel.Render("State")+": "+s.CurrentState.Render(string(branchInfo.State))) + "\n\n")

		fmt.Printf(s.Tip.Render("Use --checkout or -c to switch to the new branch") + "\n")
		fmt.Printf(s.Tip.Render("Run 'vers checkout "+branchName+"' to switch to this branch") + "\n")

		return nil
	},
}

// Helper function to get the current branch name
func getCurrentBranchName(versDir string) string {
	headFile := filepath.Join(versDir, "HEAD")
	headData, err := os.ReadFile(headFile)
	if err != nil {
		return "unknown"
	}

	headContent := string(bytes.TrimSpace(headData))
	if strings.HasPrefix(headContent, "ref: ") {
		refPath := strings.TrimPrefix(headContent, "ref: ")
		return strings.TrimPrefix(refPath, "refs/heads/")
	}

	// Detached HEAD state
	return "detached HEAD"
}

func init() {
	rootCmd.AddCommand(branchCmd)

	// Define flags for the branch command
	branchCmd.Flags().StringVarP(&fromBranch, "from", "f", "", "Source branch or commit (default: current state)")
	branchCmd.Flags().StringVarP(&branchName, "name", "n", "", "Name for the new branch")
	branchCmd.Flags().BoolP("checkout", "c", false, "Checkout the new branch after creation")
}
