package cmd

import (
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

var (
	force     bool
	isCluster bool
)

// killCmd represents the kill command
var killCmd = &cobra.Command{
	Use:   "kill [vm-id|vm-alias|cluster-id|cluster-alias]",
	Short: "Delete a VM or cluster",
	Long: `Delete a VM or cluster by ID or alias. Use -c flag for clusters.
	
Examples:
  vers kill vm-123abc          # Delete VM by ID
  vers kill my-dev-vm          # Delete VM by alias
  vers kill -c cluster-456def  # Delete cluster by ID
  vers kill -c my-cluster      # Delete cluster by alias`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		target := args[0]
		s := styles.NewKillStyles()

		// Initialize context
		baseCtx := context.Background()
		apiCtx, cancel := context.WithTimeout(baseCtx, 30*time.Second)
		defer cancel()

		if isCluster {
			return deleteCluster(apiCtx, target, &s)
		} else {
			return deleteVM(apiCtx, target, &s)
		}
	},
}

func deleteCluster(ctx context.Context, target string, s *styles.KillStyles) error {
	// Get cluster info for confirmation
	if !force {
		response, err := client.API.Cluster.Get(ctx, target)
		if err != nil {
			return fmt.Errorf(s.Error.Render("failed to get cluster information: %w"), err)
		}
		cluster := response.Data

		// Show warning with cluster details
		displayName := cluster.Alias
		if displayName == "" {
			displayName = cluster.ID
		}

		fmt.Printf(s.Warning.Render("⚠ Warning: You are about to delete cluster '%s' containing %d VMs\n"),
			displayName, cluster.VmCount)

		// Ask for confirmation
		if !askConfirmation() {
			fmt.Println(s.NoData.Render("Operation cancelled"))
			return nil
		}
	}

	// Check if this will affect HEAD before deletion
	if headWarning := checkHeadImpactSimple(target, true); headWarning != "" && !force {
		fmt.Printf(s.Warning.Render("⚠ Warning: %s\n"), headWarning)
		if !askConfirmation() {
			fmt.Println(s.NoData.Render("Operation cancelled"))
			return nil
		}
	}

	fmt.Printf(s.Progress.Render("Deleting cluster '%s'...\n"), target)

	err := client.API.Cluster.Delete(ctx, target)
	if err != nil {
		return fmt.Errorf(s.Error.Render("failed to delete cluster: %w"), err)
	}

	fmt.Printf(s.Success.Render("✓ Cluster '%s' deleted successfully\n"), target)

	// Clean up HEAD if it was pointing to a VM in this cluster
	cleanupHeadAfterDeletion()

	return nil
}

func deleteVM(ctx context.Context, target string, s *styles.KillStyles) error {
	// Show confirmation for VM deletion
	if !force {
		fmt.Printf(s.Warning.Render("⚠ Warning: You are about to delete VM '%s'\n"), target)

		if !askConfirmation() {
			fmt.Println(s.NoData.Render("Operation cancelled"))
			return nil
		}
	}

	// Check if this will affect HEAD before deletion
	if headWarning := checkHeadImpactSimple(target, false); headWarning != "" && !force {
		fmt.Printf(s.Warning.Render("⚠ Warning: %s\n"), headWarning)
		if !askConfirmation() {
			fmt.Println(s.NoData.Render("Operation cancelled"))
			return nil
		}
	}

	if force {
		fmt.Printf(s.Progress.Render("Force deleting VM '%s'...\n"), target)
	} else {
		fmt.Printf(s.Progress.Render("Deleting VM '%s'...\n"), target)
	}

	deleteParams := vers.APIVmDeleteParams{
		Recursive: vers.F(force),
	}

	response, err := client.API.Vm.Delete(ctx, target, deleteParams)
	if err != nil {
		return fmt.Errorf(s.Error.Render("failed to delete VM: %w"), err)
	}

	vm := response.Data
	displayName := vm.Alias
	if displayName == "" {
		displayName = vm.ID
	}

	fmt.Printf(s.Success.Render("✓ VM '%s' deleted successfully\n"), displayName)

	// Clean up HEAD if it was pointing to this VM
	cleanupHeadAfterDeletion()

	return nil
}

// checkHeadImpactSimple checks if deletion will affect HEAD
func checkHeadImpactSimple(target string, isCluster bool) string {
	versDir := ".vers"
	headFile := filepath.Join(versDir, "HEAD")

	if _, err := os.Stat(headFile); os.IsNotExist(err) {
		return "" // No HEAD to worry about
	}

	headData, err := os.ReadFile(headFile)
	if err != nil {
		return ""
	}

	headContent := strings.TrimSpace(string(headData))
	if headContent == "" {
		return ""
	}

	if isCluster {
		// For cluster deletion, check if HEAD VM is in the cluster
		baseCtx := context.Background()
		apiCtx, cancel := context.WithTimeout(baseCtx, 5*time.Second)
		defer cancel()

		vmResponse, err := client.API.Vm.Get(apiCtx, headContent)
		if err == nil && vmResponse.Data.ClusterID == target {
			return "Current HEAD points to a VM in the cluster being deleted. HEAD will be cleared."
		}
	} else {
		// For VM deletion, check if HEAD points to this VM
		if headContent == target {
			return "Current HEAD points to the VM being deleted. HEAD will be cleared."
		}
	}

	return ""
}

// cleanupHeadAfterDeletion clears HEAD if the VM it points to no longer exists
func cleanupHeadAfterDeletion() {
	versDir := ".vers"
	headFile := filepath.Join(versDir, "HEAD")

	headData, err := os.ReadFile(headFile)
	if err != nil {
		return
	}

	headContent := strings.TrimSpace(string(headData))
	if headContent == "" {
		return
	}

	// Check if the VM still exists
	baseCtx := context.Background()
	apiCtx, cancel := context.WithTimeout(baseCtx, 5*time.Second)
	defer cancel()

	_, err = client.API.Vm.Get(apiCtx, headContent)
	if err != nil {
		// VM no longer exists, clear HEAD
		os.WriteFile(headFile, []byte(""), 0644)
		fmt.Printf("HEAD cleared (VM no longer exists)\n")
	}
}

func askConfirmation() bool {
	fmt.Print("Are you sure you want to proceed? [y/N]: ")
	var input string
	fmt.Scanln(&input)
	return strings.EqualFold(input, "y") || strings.EqualFold(input, "yes")
}

func init() {
	rootCmd.AddCommand(killCmd)

	// Define flags for the kill command
	killCmd.Flags().BoolVarP(&force, "force", "f", false, "Force termination without confirmation")
	killCmd.Flags().BoolVarP(&isCluster, "cluster", "c", false, "Target is a cluster instead of a VM")
}
