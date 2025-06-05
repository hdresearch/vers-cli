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
			// Handle cluster deletion
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

			// Check if deleting this cluster will affect HEAD before deletion
			headWarning := checkHeadImpact(targetID)
			if headWarning != "" && !force {
				fmt.Printf(s.Warning.Render("⚠ Warning: %s\n"), headWarning)
				fmt.Print("Do you want to continue? [y/N]: ")
				var input string
				fmt.Scanln(&input)
				if !strings.EqualFold(input, "y") && !strings.EqualFold(input, "yes") {
					fmt.Println(s.NoData.Render("Operation cancelled"))
					return nil
				}
			}

			_, err := client.API.Cluster.Delete(apiCtx, targetID)
			if err != nil {
				return fmt.Errorf(s.Error.Render("failed to delete cluster: %w"), err)
			}
			fmt.Printf(s.Success.Render("✓ Cluster '%s' deleted successfully\n"), targetID)

			// Clean up local repository state after successful cluster deletion
			if err := cleanupAfterClusterDeletion(); err != nil {
				fmt.Printf(s.Warning.Render("Warning: %s\n"), err)
			}

		} else {
			// Handle VM deletion with confirmation system
			if !force {
				fmt.Printf(s.Warning.Render("⚠ Warning: You are about to delete VM '%s'\n"), targetID)
				fmt.Print("Are you sure you want to proceed? [y/N]: ")
				var input string
				fmt.Scanln(&input)

				if !strings.EqualFold(input, "y") && !strings.EqualFold(input, "yes") {
					fmt.Println(s.NoData.Render("Operation cancelled"))
					return nil
				}
			}

			if force {
				fmt.Printf(s.Progress.Render("Force deleting VM '%s'...\n"), targetID)
			} else {
				fmt.Printf(s.Progress.Render("Deleting VM '%s'...\n"), targetID)
			}

			// Check if deleting this VM will affect HEAD before deletion
			headWarning := checkVMHeadImpact(targetID)
			if headWarning != "" && !force {
				fmt.Printf(s.Warning.Render("⚠ Warning: %s\n"), headWarning)
				fmt.Print("Do you want to continue? [y/N]: ")
				var input string
				fmt.Scanln(&input)
				if !strings.EqualFold(input, "y") && !strings.EqualFold(input, "yes") {
					fmt.Println(s.NoData.Render("Operation cancelled"))
					return nil
				}
			}

			deleteParams := vers.APIVmDeleteParams{
				Recursive: vers.F(force),
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

	// Update HEAD if necessary - always set to detached state
	if needsHeadUpdate {
		detachedMessage := "DETACHED_HEAD"
		if err := os.WriteFile(headFile, []byte(detachedMessage+"\n"), 0644); err != nil {
			return fmt.Errorf("failed to set HEAD to detached state")
		}
		fmt.Printf("HEAD is now in detached state (VM deletion affected current branch)\n")
	}

	// Report cleanup results
	if len(branchesDeleted) > 0 {
		fmt.Printf("Cleaned up branches pointing to deleted VM: %s\n", strings.Join(branchesDeleted, ", "))
	}

	return nil
}

// cleanupAfterClusterDeletion handles local repository cleanup after a cluster is deleted
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

	// Update HEAD if necessary - always set to detached state
	if needsHeadUpdate {
		detachedMessage := "DETACHED_HEAD"
		if err := os.WriteFile(headFile, []byte(detachedMessage+"\n"), 0644); err != nil {
			return fmt.Errorf("failed to set HEAD to detached state")
		}
		fmt.Printf("HEAD is now in detached state (cluster deletion affected current branch)\n")
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

// checkHeadImpact checks if deleting a cluster will affect the current HEAD
func checkHeadImpact(clusterID string) string {
	versDir := ".vers"
	headFile := filepath.Join(versDir, "HEAD")

	// Check if .vers directory exists
	if _, err := os.Stat(versDir); os.IsNotExist(err) {
		return "" // No local repo
	}

	// Read current HEAD
	headData, err := os.ReadFile(headFile)
	if err != nil {
		return ""
	}

	headContent := string(bytes.TrimSpace(headData))

	// If HEAD points to a branch, check if that branch's VM is in the cluster being deleted
	if strings.HasPrefix(headContent, "ref: ") {
		refPath := strings.TrimPrefix(headContent, "ref: ")
		branchName := strings.TrimPrefix(refPath, "refs/heads/")

		// Read the branch to get its VM ID
		branchPath := filepath.Join(versDir, refPath)
		branchData, err := os.ReadFile(branchPath)
		if err != nil {
			return ""
		}

		branchVMID := string(bytes.TrimSpace(branchData))

		// Check if this VM is in the cluster being deleted
		baseCtx := context.Background()
		apiCtx, cancel := context.WithTimeout(baseCtx, 5*time.Second)
		defer cancel()

		response, err := client.API.Vm.Get(apiCtx, branchVMID)
		if err == nil && response.Data.ClusterID == clusterID {
			return fmt.Sprintf("Current branch '%s' points to a VM in the cluster being deleted. HEAD will become detached.", branchName)
		}
	}

	return ""
}

// checkVMHeadImpact checks if deleting a VM will affect the current HEAD
func checkVMHeadImpact(vmID string) string {
	versDir := ".vers"
	headFile := filepath.Join(versDir, "HEAD")

	// Check if .vers directory exists
	if _, err := os.Stat(versDir); os.IsNotExist(err) {
		return "" // No local repo
	}

	// Read current HEAD
	headData, err := os.ReadFile(headFile)
	if err != nil {
		return ""
	}

	headContent := string(bytes.TrimSpace(headData))

	// Check if HEAD points directly to the VM being deleted
	if headContent == vmID {
		return "Current HEAD points directly to the VM being deleted. HEAD will become detached."
	}

	// If HEAD points to a branch, check if that branch points to the VM being deleted
	if strings.HasPrefix(headContent, "ref: ") {
		refPath := strings.TrimPrefix(headContent, "ref: ")
		branchName := strings.TrimPrefix(refPath, "refs/heads/")

		// Read the branch to get its VM ID
		branchPath := filepath.Join(versDir, refPath)
		branchData, err := os.ReadFile(branchPath)
		if err != nil {
			return ""
		}

		branchVMID := string(bytes.TrimSpace(branchData))

		// Check if this branch points to the VM being deleted
		if branchVMID == vmID {
			return fmt.Sprintf("Current branch '%s' points to the VM being deleted. HEAD will become detached.", branchName)
		}
	}

	return ""
}

func init() {
	rootCmd.AddCommand(killCmd)
	killCmd.Flags().BoolVarP(&force, "force", "f", false, "Force termination without confirmation")
	killCmd.Flags().BoolVarP(&isCluster, "cluster", "c", false, "Target is a cluster instead of a VM")
}
