package cmd

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	vers "github.com/hdresearch/vers-sdk-go"
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

		// If no VM ID provided, try to use the current HEAD
		if len(args) == 0 {
			// Read the HEAD file to find the current branch
			versDir := ".vers"
			headFile := filepath.Join(versDir, "HEAD")

			// Check if .vers directory and HEAD file exist
			if _, err := os.Stat(headFile); os.IsNotExist(err) {
				return fmt.Errorf("no VM ID provided and HEAD not found. Run 'vers init' first")
			}

			// Read HEAD file
			headData, err := os.ReadFile(headFile)
			if err != nil {
				return fmt.Errorf("error reading HEAD: %w", err)
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
					return fmt.Errorf("error reading reference '%s': %w", refPath, err)
				}

				// Get the VM ID from the reference file
				vmName = string(bytes.TrimSpace(refData))
			} else {
				// HEAD directly contains a VM ID
				vmName = headContent
			}

			if vmName == "" {
				return fmt.Errorf("could not determine current VM ID from HEAD")
			}

			fmt.Printf("Using current HEAD VM: %s\n", vmName)
		} else {
			vmName = args[0]
		}

		// We'll set the final branch name after we have the new VM ID
		// Temporary placeholder for branch name display
		tempBranchName := branchName
		if tempBranchName == "" {
			tempBranchName = "[auto-generated]"
		}

		fmt.Printf("Creating branch of vm '%s' as '%s'\n", vmName, tempBranchName)

		baseCtx := context.Background()
		client = vers.NewClient()
		apiCtx, cancel := context.WithTimeout(baseCtx, 30*time.Second)
		defer cancel()

		branchParams := vers.APIVmNewBranchParams{
			Body: map[string]interface{}{},
		}

		fmt.Println("Creating branch...")
		branchInfo, err := client.API.Vm.NewBranch(apiCtx, vmName, branchParams)

		if err != nil {
			return fmt.Errorf("failed to create branch of vm '%s': %w", vmName, err)
		}
		fmt.Printf("Branch created successfully with ID: %s\n", branchInfo.ID)
		fmt.Printf("Branch IP address: %s\n", branchInfo.IPAddress)
		fmt.Printf("Branch state: %s\n", branchInfo.State)

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
				fmt.Println("Warning: .vers directory not found. Run 'vers init' first.")
			} else {
				// Sanitize branch name (remove invalid characters for filenames)
				safeBranchName := strings.ReplaceAll(branchName, "/", "-")
				safeBranchName = strings.ReplaceAll(safeBranchName, "\\", "-")

				// Create branch ref file
				branchRefPath := filepath.Join(versDir, "refs", "heads", safeBranchName)
				if err := os.WriteFile(branchRefPath, []byte(branchVmID+"\n"), 0644); err != nil {
					fmt.Printf("Warning: Failed to create branch ref: %v\n", err)
				} else {
					fmt.Printf("Created branch reference: refs/heads/%s -> %s\n", safeBranchName, branchVmID)
				}

				// Optionally, switch HEAD to the new branch
				if checkout, _ := cmd.Flags().GetBool("checkout"); checkout {
					headFile := filepath.Join(versDir, "HEAD")
					newRef := fmt.Sprintf("ref: refs/heads/%s\n", safeBranchName)
					if err := os.WriteFile(headFile, []byte(newRef), 0644); err != nil {
						fmt.Printf("Warning: Failed to update HEAD: %v\n", err)
					} else {
						fmt.Printf("HEAD now points to: refs/heads/%s\n", safeBranchName)
					}
				} else {
					// Show message indicating HEAD was not changed
					currentBranch := getCurrentBranchName(versDir)
					fmt.Printf("Note: HEAD is still on '%s'. Use --checkout or -c to switch to the new branch.\n", currentBranch)
					fmt.Printf("Run 'vers checkout %s' to switch to this branch.\n", safeBranchName)
				}
			}
		}

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
