package cmd

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/hdresearch/vers-cli/internal/utils"
	"github.com/hdresearch/vers-cli/styles"
	vers "github.com/hdresearch/vers-sdk-go"
	"github.com/spf13/cobra"
)

var (
	force     bool
	isCluster bool
	killAll   bool
)

// killCmd represents the kill command
var killCmd = &cobra.Command{
	Use:   "kill [vm-id|vm-alias|cluster-id|cluster-alias]",
	Short: "Delete a VM or cluster",
	Long: `Delete a VM or cluster by ID or alias. Use -c flag for clusters, or -a flag to delete all clusters.
	
Examples:
  vers kill vm-123abc          # Delete VM by ID
  vers kill my-dev-vm          # Delete VM by alias
  vers kill -c cluster-456def  # Delete cluster by ID
  vers kill -c my-cluster      # Delete cluster by alias
  vers kill -a                 # Delete ALL clusters (use with caution!)
  vers kill -a --force         # Delete ALL clusters without confirmation`,
	Args: func(cmd *cobra.Command, args []string) error {
		// If --all flag is used, no arguments should be provided
		if killAll {
			if len(args) > 0 {
				return fmt.Errorf("cannot specify target when using --all flag")
			}
			return nil
		}
		// Otherwise, exactly one argument is required
		return cobra.ExactArgs(1)(cmd, args)
	},
	RunE: func(cmd *cobra.Command, args []string) error {
		s := styles.NewKillStyles()

		// Initialize context
		baseCtx := context.Background()
		apiCtx, cancel := context.WithTimeout(baseCtx, 60*time.Second) // Longer timeout for bulk operations
		defer cancel()

		// Handle kill all clusters
		if killAll {
			return deleteAllClusters(apiCtx, &s)
		}

		// Handle single target deletion
		target := args[0]
		if isCluster {
			return deleteCluster(apiCtx, target, &s)
		} else {
			return deleteVM(apiCtx, target, &s)
		}
	},
}

func deleteAllClusters(ctx context.Context, s *styles.KillStyles) error {
	fmt.Println(s.Progress.Render("Fetching all clusters..."))

	// Get list of all clusters
	response, err := client.API.Cluster.List(ctx)
	if err != nil {
		return fmt.Errorf(s.Error.Render("failed to list clusters: %w"), err)
	}
	clusters := response.Data

	if len(clusters) == 0 {
		fmt.Println(s.NoData.Render("No clusters found to delete."))
		return nil
	}

	// Show warning about what will be deleted
	if !force {
		fmt.Printf(s.Warning.Render("⚠️  DANGER: You are about to delete ALL %d clusters and their VMs:\n\n"), len(clusters))

		for i, cluster := range clusters {
			displayName := cluster.Alias
			if displayName == "" {
				displayName = cluster.ID
			}
			fmt.Printf(s.Warning.Render("  %d. Cluster '%s' (%d VMs)\n"), i+1, displayName, cluster.VmCount)
		}

		fmt.Print(s.Warning.Render("\n⚠️  This action is IRREVERSIBLE and will delete ALL your data!\n"))

		// Ask for explicit confirmation
		fmt.Print(s.Warning.Render("Type 'DELETE ALL' to confirm: "))
		var input string
		fmt.Scanln(&input)

		if input != "DELETE ALL" {
			fmt.Println(s.NoData.Render("Operation cancelled - input did not match 'DELETE ALL'"))
			return nil
		}
	}

	fmt.Printf(s.Progress.Render("Deleting %d clusters...\n"), len(clusters))

	// Track results
	var successCount, failCount int
	var errors []string

	// Delete each cluster
	for i, cluster := range clusters {
		displayName := cluster.Alias
		if displayName == "" {
			displayName = cluster.ID
		}

		fmt.Printf(s.Progress.Render("[%d/%d] Deleting cluster '%s'...\n"), i+1, len(clusters), displayName)

		result, err := client.API.Cluster.Delete(ctx, cluster.ID)
		if err != nil {
			failCount++
			errorMsg := fmt.Sprintf("Cluster '%s': %v", displayName, err)
			errors = append(errors, errorMsg)
			fmt.Printf(s.Error.Render("  ❌ Failed: %s\n"), err)
			continue
		}

		// Check for partial failures using util file
		if errorSummary := utils.GetClusterDeleteErrorSummary(result); errorSummary != "" {
			failCount++
			errorMsg := fmt.Sprintf("Cluster '%s' partially failed: %s", displayName, errorSummary)
			errors = append(errors, errorMsg)
			fmt.Printf(s.Warning.Render("  ⚠️  Partially failed: %s\n"), errorSummary)
		} else {
			successCount++
			fmt.Print(s.Success.Render("  ✓ Deleted successfully\n"))
		}
	}

	// Summary
	fmt.Print(s.Progress.Render("\n=== Deletion Summary ===\n"))
	fmt.Printf(s.Success.Render("✓ Successfully deleted: %d clusters\n"), successCount)

	if failCount > 0 {
		fmt.Printf(s.Error.Render("❌ Failed to delete: %d clusters\n"), failCount)
		fmt.Print(s.Warning.Render("\nError details:\n"))
		for _, error := range errors {
			fmt.Printf(s.Warning.Render("  • %s\n"), error)
		}
	}

	// Clean up HEAD since we deleted everything
	if successCount > 0 {
		cleanupHeadAfterDeletion()
		fmt.Print(s.NoData.Render("\nHEAD cleared (all clusters deleted)\n"))
	}

	if failCount > 0 {
		return fmt.Errorf("some clusters failed to delete - see details above")
	}

	fmt.Print(s.Success.Render("\n🎉 All clusters deleted successfully!\n"))
	return nil
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

	// Use the improved API call with detailed error handling from main branch
	result, err := client.API.Cluster.Delete(ctx, target)
	if err != nil {
		return fmt.Errorf(s.Error.Render("failed to delete cluster: %w"), err)
	}

	// Handle errors using utility
	if utils.HandleClusterDeleteErrors(result, s) {
		// Errors were printed by the utility
	} else {
		fmt.Printf(s.Success.Render("✓ Cluster '%s' deleted successfully\n"), result.Data.ClusterID)
	}

	// Clean up HEAD if it was pointing to a VM in this cluster (your simplified approach)
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

	// Use the improved API call with detailed error handling from main branch
	result, err := client.API.Vm.Delete(ctx, target, deleteParams)
	if err != nil {
		return fmt.Errorf(s.Error.Render("failed to delete VM: %w"), err)
	}

	// Handle errors using utility
	if utils.HandleVmDeleteErrors(result, s) {
		// Errors were printed by the utility
	} else {
		fmt.Printf(s.Success.Render("✓ VM '%s' deleted successfully\n"), target)
	}

	// Clean up HEAD if it was pointing to this VM (your simplified approach)
	cleanupHeadAfterDeletion()

	return nil
}

// checkHeadImpactSimple checks if deletion will affect HEAD (your simplified approach)
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

// cleanupHeadAfterDeletion clears HEAD if the VM it points to no longer exists (your simplified approach)
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
	killCmd.Flags().BoolVarP(&killAll, "all", "a", false, "Delete ALL clusters (use with extreme caution)")
}
