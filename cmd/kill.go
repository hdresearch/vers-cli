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
	vers "github.com/hdresearch/vers-sdk-go"
	"github.com/spf13/cobra"
)

var force bool
var isCluster bool

// killCmd represents the kill command
var killCmd = &cobra.Command{
	Use:   "kill [-c] <vm-id|cluster-id>",
	Short: "Forcefully terminate a VM or cluster",
	Long:  `Forcefully terminate a VM or cluster in the Vers environment. Use -c flag to specify a cluster.`,
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		targetID := args[0]
		s := styles.NewKillStyles()

		// Initialize SDK client and context
		baseCtx := context.Background()
		apiCtx, cancel := context.WithTimeout(baseCtx, 30*time.Second)
		defer cancel()

		if isCluster {
			// Handle cluster deletion (existing code)
			if !force {
				response, err := client.API.Cluster.Get(apiCtx, targetID)
				if err != nil {
					return fmt.Errorf(s.Error.Render("failed to get cluster information: %w"), err)
				}
				cluster := response.Data

				fmt.Printf(s.Warning.Render("⚠ Warning: You are about to delete cluster '%s' containing %d VMs\n"),
					targetID, cluster.VmCount)

				fmt.Print("Are you sure you want to proceed? [y/N]: ")
				var input string
				fmt.Scanln(&input)

				if !strings.EqualFold(input, "y") && !strings.EqualFold(input, "yes") {
					fmt.Println(s.NoData.Render("Operation cancelled"))
					return nil
				}
			}

			fmt.Printf(s.Progress.Render("Deleting cluster '%s'...\n"), targetID)
			_, err := client.API.Cluster.Delete(apiCtx, targetID)
			if err != nil {
				return fmt.Errorf(s.Error.Render("failed to delete cluster: %w"), err)
			}
			fmt.Printf(s.Success.Render("✓ Cluster '%s' deleted successfully\n"), targetID)

			// Clean up local repository state after successful cluster deletion
			// Since cluster deletion removes all VMs in the cluster, we need to clean up
			// any branches that pointed to those VMs
			if err := cleanupAfterClusterDeletion(); err != nil {
				fmt.Printf(s.Warning.Render("Warning: %s\n"), err)
			}

		} else {
			// Handle VM deletion
			if force {
				fmt.Printf(s.Progress.Render("Force deleting VM '%s'...\n"), targetID)
			} else {
				fmt.Printf(s.Progress.Render("Deleting VM '%s'...\n"), targetID)
			}

			deleteParams := vers.APIVmDeleteParams{
				Recursive: vers.F(false),
			}
			response, err := client.API.Vm.Delete(apiCtx, targetID, deleteParams)
			if err != nil {
				return fmt.Errorf(s.Error.Render("failed to delete VM: %w"), err)
			}
			vm := response.Data
			fmt.Printf(s.Success.Render("✓ VM '%s' deleted successfully\n"), vm.ID)

			// Clean up local repository state after successful VM deletion
			if err := cleanupAfterVMDeletion(targetID); err != nil {
				fmt.Printf(s.Warning.Render("Warning: %s\n"), err)
			}
		}

		return nil
	},
}

// cleanupAfterVMDeletion handles local repository cleanup after a VM is deleted
func cleanupAfterVMDeletion(vmID string) error {
	versDir := ".vers"
	headFile := filepath.Join(versDir, "HEAD")

	// Check if .vers directory exists
	if _, err := os.Stat(versDir); os.IsNotExist(err) {
		return nil // No local repo to clean up
	}

	// Find and delete any branches pointing to this VM
	branchesDeleted := []string{}
	refsHeadsDir := filepath.Join(versDir, "refs", "heads")
	if _, err := os.Stat(refsHeadsDir); err == nil {
		entries, err := os.ReadDir(refsHeadsDir)
		if err == nil {
			for _, entry := range entries {
				if entry.IsDir() {
					continue
				}

				branchPath := filepath.Join(refsHeadsDir, entry.Name())
				branchData, err := os.ReadFile(branchPath)
				if err != nil {
					continue
				}

				branchVMID := string(bytes.TrimSpace(branchData))
				if branchVMID == vmID {
					// This branch points to the deleted VM
					if err := os.Remove(branchPath); err == nil {
						branchesDeleted = append(branchesDeleted, entry.Name())
					}
				}
			}
		}
	}

	// Check if HEAD points to the deleted VM (either directly or via a deleted branch)
	needsHeadUpdate := false
	headData, err := os.ReadFile(headFile)
	if err == nil {
		headContent := string(bytes.TrimSpace(headData))

		if headContent == vmID {
			// HEAD points directly to the deleted VM
			needsHeadUpdate = true
		} else if strings.HasPrefix(headContent, "ref: ") {
			// HEAD points to a branch - check if that branch was deleted
			refPath := strings.TrimPrefix(headContent, "ref: ")
			branchName := strings.TrimPrefix(refPath, "refs/heads/")
			for _, deletedBranch := range branchesDeleted {
				if deletedBranch == branchName {
					needsHeadUpdate = true
					break
				}
			}
		}
	}

	// Update HEAD if necessary
	if needsHeadUpdate {
		// Try to find another branch to switch to
		if newBranch := findAvailableBranch(versDir); newBranch != "" {
			fmt.Printf("DEBUG: Switching HEAD to branch '%s'\n", newBranch)
			newRef := fmt.Sprintf("ref: refs/heads/%s", newBranch)
			if err := os.WriteFile(headFile, []byte(newRef+"\n"), 0644); err != nil {
				return fmt.Errorf("failed to update HEAD to branch '%s'", newBranch)
			}
			fmt.Printf("Switched to branch '%s'\n", newBranch)
		} else {
			fmt.Printf("DEBUG: No branches available for HEAD assignment\n")
			// No branches available - set HEAD to a placeholder
			placeholder := "# No branches available - run 'vers checkout -c <branch-name>' to create one"
			if err := os.WriteFile(headFile, []byte(placeholder+"\n"), 0644); err != nil {
				return fmt.Errorf("failed to update HEAD")
			}
			fmt.Println("No branches available. Create a new branch with 'vers checkout -c <branch-name>'")
		}
	} else {
		fmt.Printf("DEBUG: HEAD does not need update\n")
	}

	// Report cleanup results
	if len(branchesDeleted) > 0 {
		fmt.Printf("Cleaned up branches pointing to deleted VM: %s\n", strings.Join(branchesDeleted, ", "))
	}

	return nil
}

// findAvailableBranch finds the first available branch in the repository
func findAvailableBranch(versDir string) string {
	refsHeadsDir := filepath.Join(versDir, "refs", "heads")
	entries, err := os.ReadDir(refsHeadsDir)
	if err != nil {
		return ""
	}

	for _, entry := range entries {
		if !entry.IsDir() {
			return entry.Name()
		}
	}
	return ""
}

// cleanupAfterClusterDeletion handles local repository cleanup after a cluster is deleted
// Since we don't know which specific VMs were in the cluster, we validate all branches
func cleanupAfterClusterDeletion() error {
	versDir := ".vers"
	headFile := filepath.Join(versDir, "HEAD")

	// Check if .vers directory exists
	if _, err := os.Stat(versDir); os.IsNotExist(err) {
		return nil // No local repo to clean up
	}

	// Find and delete any branches pointing to non-existent VMs
	branchesDeleted := []string{}
	refsHeadsDir := filepath.Join(versDir, "refs", "heads")
	if _, err := os.Stat(refsHeadsDir); err == nil {
		entries, err := os.ReadDir(refsHeadsDir)
		if err == nil {
			for _, entry := range entries {
				if entry.IsDir() {
					continue
				}

				branchPath := filepath.Join(refsHeadsDir, entry.Name())
				branchData, err := os.ReadFile(branchPath)
				if err != nil {
					continue
				}

				branchVMID := string(bytes.TrimSpace(branchData))

				// Check if this VM still exists
				if !vmExists(branchVMID) {
					// This branch points to a deleted VM
					if err := os.Remove(branchPath); err == nil {
						branchesDeleted = append(branchesDeleted, entry.Name())
					}
				}
			}
		}
	}

	// Check if HEAD points to a deleted branch or non-existent VM
	needsHeadUpdate := false
	headData, err := os.ReadFile(headFile)
	if err == nil {
		headContent := string(bytes.TrimSpace(headData))

		if strings.HasPrefix(headContent, "vm-") && !vmExists(headContent) {
			// HEAD points directly to a deleted VM
			needsHeadUpdate = true
		} else if strings.HasPrefix(headContent, "ref: ") {
			// HEAD points to a branch - check if that branch was deleted
			refPath := strings.TrimPrefix(headContent, "ref: ")
			branchName := strings.TrimPrefix(refPath, "refs/heads/")
			for _, deletedBranch := range branchesDeleted {
				if deletedBranch == branchName {
					needsHeadUpdate = true
					break
				}
			}
		}
	}

	// Update HEAD if necessary
	if needsHeadUpdate {
		// Try to find another branch to switch to
		if newBranch := findAvailableBranch(versDir); newBranch != "" {
			newRef := fmt.Sprintf("ref: refs/heads/%s", newBranch)
			if err := os.WriteFile(headFile, []byte(newRef+"\n"), 0644); err != nil {
				return fmt.Errorf("failed to update HEAD to branch '%s'", newBranch)
			}
			fmt.Printf("Switched to branch '%s'\n", newBranch)
		} else {
			// No branches available - set HEAD to a placeholder
			placeholder := "# No branches available - run 'vers checkout -c <branch-name>' to create one"
			if err := os.WriteFile(headFile, []byte(placeholder+"\n"), 0644); err != nil {
				return fmt.Errorf("failed to update HEAD")
			}
			fmt.Println("No branches available. Create a new branch with 'vers checkout -c <branch-name>'")
		}
	}

	// Report cleanup results
	if len(branchesDeleted) > 0 {
		fmt.Printf("Cleaned up branches pointing to deleted VMs: %s\n", strings.Join(branchesDeleted, ", "))
	}

	return nil
}

// vmExists checks if a VM ID still exists by attempting to fetch it
func vmExists(vmID string) bool {
	if vmID == "" {
		return false
	}

	// Use a short timeout for VM existence checks
	baseCtx := context.Background()
	apiCtx, cancel := context.WithTimeout(baseCtx, 5*time.Second)
	defer cancel()

	_, err := client.API.Vm.Get(apiCtx, vmID)
	return err == nil
}

func init() {
	rootCmd.AddCommand(killCmd)
	killCmd.Flags().BoolVarP(&force, "force", "f", false, "Force termination without confirmation")
	killCmd.Flags().BoolVarP(&isCluster, "cluster", "c", false, "Target is a cluster instead of a VM")
}
