package cmd

import (
	"bufio"
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
	Use:   "kill [vm-id|vm-alias|cluster-id|cluster-alias]...",
	Short: "Delete one or more VMs or clusters",
	Long: `Delete one or more VMs or clusters by ID or alias. Use -c flag for clusters, or -a flag to delete all clusters.
	
Examples:
  vers kill vm-123abc                    # Delete single VM by ID
  vers kill my-dev-vm my-test-vm         # Delete multiple VMs by alias
  vers kill -c cluster-456def            # Delete single cluster by ID
  vers kill -c my-cluster other-cluster  # Delete multiple clusters by alias
  vers kill -a                           # Delete ALL clusters (use with caution!)
  vers kill -a --force                   # Delete ALL clusters without confirmation`,
	Args: func(cmd *cobra.Command, args []string) error {
		// If --all flag is used, no arguments should be provided
		if killAll {
			if len(args) > 0 {
				return fmt.Errorf("cannot specify target when using --all flag")
			}
			return nil
		}
		// Otherwise, at least one argument is required
		if len(args) == 0 {
			return fmt.Errorf("requires at least 1 arg(s), received 0")
		}
		return nil
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

		// Handle multiple target deletion
		targets := args
		if isCluster {
			return deleteMultipleClusters(apiCtx, targets, &s)
		} else {
			return deleteMultipleVMs(apiCtx, targets, &s)
		}
	},
}

func deleteMultipleClusters(ctx context.Context, targets []string, s *styles.KillStyles) error {
	if len(targets) == 1 {
		// Single cluster - use existing logic
		return deleteCluster(ctx, targets[0], s)
	}

	// Multiple clusters
	fmt.Printf(s.Progress.Render("Processing %d clusters for deletion...\n"), len(targets))

	// Validate all clusters exist first (if not force)
	var clustersToDelete []struct {
		Target      string
		DisplayName string
		VmCount     int
	}

	if !force {
		for _, target := range targets {
			response, err := client.API.Cluster.Get(ctx, target)
			if err != nil {
				return fmt.Errorf(s.Error.Render("failed to get cluster information for '%s': %w"), target, err)
			}
			cluster := response.Data

			displayName := cluster.Alias
			if displayName == "" {
				displayName = cluster.ID
			}

			clustersToDelete = append(clustersToDelete, struct {
				Target      string
				DisplayName string
				VmCount     int
			}{
				Target:      target,
				DisplayName: displayName,
				VmCount:     int(cluster.VmCount),
			})
		}

		// Show warning about what will be deleted
		fmt.Printf(s.Warning.Render("Warning: You are about to delete %d clusters:\n"), len(clustersToDelete))
		fmt.Println()

		for i, cluster := range clustersToDelete {
			listItem := fmt.Sprintf("  %d. Cluster '%s' (%d VMs)", i+1, cluster.DisplayName, cluster.VmCount)
			fmt.Println(s.Warning.Render(listItem))
		}

		fmt.Println()
		fmt.Println(s.Warning.Render("This action is IRREVERSIBLE and will delete ALL specified clusters and their data!"))

		// Ask for confirmation
		if !askConfirmation() {
			fmt.Println(s.NoData.Render("Operation cancelled"))
			return nil
		}
	}

	// Check for HEAD impact
	headImpacted := false
	for _, target := range targets {
		if headWarning := checkHeadImpactSimple(target, true); headWarning != "" && !force {
			if !headImpacted {
				fmt.Println(s.Warning.Render("Warning: Some clusters contain the current HEAD VM"))
				headImpacted = true
			}
		}
	}

	if headImpacted && !force {
		if !askConfirmation() {
			fmt.Println(s.NoData.Render("Operation cancelled"))
			return nil
		}
	}

	// Delete each cluster
	var successCount, failCount int
	var errors []string

	for i, target := range targets {
		displayName := target
		if !force && i < len(clustersToDelete) {
			displayName = clustersToDelete[i].DisplayName
		}

		clusterProgressMsg := fmt.Sprintf("[%d/%d] Deleting cluster '%s'...", i+1, len(targets), displayName)
		fmt.Println(s.Progress.Render(clusterProgressMsg))

		result, err := client.API.Cluster.Delete(ctx, target)
		if err != nil {
			failCount++
			errorMsg := fmt.Sprintf("Cluster '%s': %v", displayName, err)
			errors = append(errors, errorMsg)

			failMsg := fmt.Sprintf("  Failed: %s", err)
			fmt.Println(s.Error.Render(failMsg))
			continue
		}

		// Check for partial failures using util file
		if errorSummary := utils.GetClusterDeleteErrorSummary(result); errorSummary != "" {
			failCount++
			errorMsg := fmt.Sprintf("Cluster '%s' partially failed: %s", displayName, errorSummary)
			errors = append(errors, errorMsg)

			partialFailMsg := fmt.Sprintf("  Partially failed: %s", errorSummary)
			fmt.Println(s.Warning.Render(partialFailMsg))
		} else {
			successCount++
			fmt.Println(s.Success.Render("  ✓ Deleted successfully"))
		}
	}

	// Summary
	fmt.Println()
	fmt.Println(s.Progress.Render("=== Deletion Summary ==="))

	successMsg := fmt.Sprintf("✓ Successfully deleted: %d clusters", successCount)
	fmt.Println(s.Success.Render(successMsg))

	if failCount > 0 {
		failMsg := fmt.Sprintf("Failed to delete: %d clusters", failCount)
		fmt.Println(s.Error.Render(failMsg))

		fmt.Println()
		fmt.Println(s.Warning.Render("Error details:"))
		for _, error := range errors {
			errorDetail := fmt.Sprintf("  • %s", error)
			fmt.Println(s.Warning.Render(errorDetail))
		}
	}

	// Clean up HEAD
	if successCount > 0 {
		cleanupHeadAfterDeletion()
		fmt.Println()
		fmt.Println(s.NoData.Render("HEAD cleared (clusters deleted)"))
	}

	if failCount > 0 {
		return fmt.Errorf("some clusters failed to delete - see details above")
	}

	if len(targets) > 1 {
		fmt.Println()
		fmt.Println(s.Success.Render("All specified clusters deleted successfully!"))
	}

	return nil
}

func deleteMultipleVMs(ctx context.Context, targets []string, s *styles.KillStyles) error {
	if len(targets) == 1 {
		// Single VM - use existing logic
		return deleteVM(ctx, targets[0], s)
	}

	// Multiple VMs
	fmt.Printf(s.Progress.Render("Processing %d VMs for deletion...\n"), len(targets))

	// Show confirmation for multiple VM deletion
	if !force {
		fmt.Printf(s.Warning.Render("Warning: You are about to delete %d VMs:\n"), len(targets))
		fmt.Println()

		for i, target := range targets {
			listItem := fmt.Sprintf("  %d. VM '%s'", i+1, target)
			fmt.Println(s.Warning.Render(listItem))
		}

		fmt.Println()
		fmt.Println(s.Warning.Render("This action is IRREVERSIBLE and will delete ALL specified VMs!"))

		if !askConfirmation() {
			fmt.Println(s.NoData.Render("Operation cancelled"))
			return nil
		}
	}

	// Check for HEAD impact
	headImpacted := false
	for _, target := range targets {
		if headWarning := checkHeadImpactSimple(target, false); headWarning != "" && !force {
			if !headImpacted {
				fmt.Println(s.Warning.Render("Warning: Some VMs are the current HEAD"))
				headImpacted = true
			}
		}
	}

	if headImpacted && !force {
		if !askConfirmation() {
			fmt.Println(s.NoData.Render("Operation cancelled"))
			return nil
		}
	}

	// Delete each VM
	var successCount, failCount int
	var errors []string

	for i, target := range targets {
		var progressMsg string
		if force {
			progressMsg = fmt.Sprintf("[%d/%d] Force deleting VM '%s'...", i+1, len(targets), target)
		} else {
			progressMsg = fmt.Sprintf("[%d/%d] Deleting VM '%s'...", i+1, len(targets), target)
		}
		fmt.Println(s.Progress.Render(progressMsg))

		deleteParams := vers.APIVmDeleteParams{
			Recursive: vers.F(force),
		}

		result, err := client.API.Vm.Delete(ctx, target, deleteParams)
		if err != nil {
			failCount++
			errorMsg := fmt.Sprintf("VM '%s': %v", target, err)
			errors = append(errors, errorMsg)

			failMsg := fmt.Sprintf("  Failed: %s", err)
			fmt.Println(s.Error.Render(failMsg))
			continue
		}

		// Handle errors using utility
		if utils.HandleVmDeleteErrors(result, s) {
			failCount++
			errorMsg := fmt.Sprintf("VM '%s': deletion had errors", target)
			errors = append(errors, errorMsg)
		} else {
			successCount++
			fmt.Println(s.Success.Render("  ✓ Deleted successfully"))
		}
	}

	// Summary
	fmt.Println()
	fmt.Println(s.Progress.Render("=== Deletion Summary ==="))

	successMsg := fmt.Sprintf("✓ Successfully deleted: %d VMs", successCount)
	fmt.Println(s.Success.Render(successMsg))

	if failCount > 0 {
		failMsg := fmt.Sprintf("Failed to delete: %d VMs", failCount)
		fmt.Println(s.Error.Render(failMsg))

		fmt.Println()
		fmt.Println(s.Warning.Render("Error details:"))
		for _, error := range errors {
			errorDetail := fmt.Sprintf("  • %s", error)
			fmt.Println(s.Warning.Render(errorDetail))
		}
	}

	// Clean up HEAD
	if successCount > 0 {
		cleanupHeadAfterDeletion()
		fmt.Println()
		fmt.Println(s.NoData.Render("HEAD cleared (VMs deleted)"))
	}

	if failCount > 0 {
		return fmt.Errorf("some VMs failed to delete - see details above")
	}

	if len(targets) > 1 {
		fmt.Println()
		fmt.Println(s.Success.Render("All specified VMs deleted successfully!"))
	}

	return nil
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
		// Format the header message first, then render and print
		headerMsg := fmt.Sprintf("DANGER: You are about to delete ALL %d clusters and their VMs:", len(clusters))
		fmt.Println(s.Warning.Render(headerMsg))
		fmt.Println() // Empty line for spacing

		for i, cluster := range clusters {
			displayName := cluster.Alias
			if displayName == "" {
				displayName = cluster.ID
			}
			// Format the list item, then render and print
			listItem := fmt.Sprintf("  %d. Cluster '%s' (%d VMs)", i+1, displayName, int(cluster.VmCount))
			fmt.Println(s.Warning.Render(listItem))
		}

		fmt.Println() // Empty line for spacing
		fmt.Println(s.Warning.Render("This action is IRREVERSIBLE and will delete ALL your data!"))
		fmt.Println()

		// Ask for explicit confirmation
		fmt.Printf(s.Warning.Render("Type 'DELETE ALL' to confirm: "))

		reader := bufio.NewReader(os.Stdin)
		input, err := reader.ReadString('\n')
		if err != nil {
			fmt.Println(s.NoData.Render("Error reading input"))
			return nil
		}
		input = strings.TrimSpace(input)

		if input != "DELETE ALL" {
			fmt.Println(s.NoData.Render("Operation cancelled - input did not match 'DELETE ALL'"))
			return nil
		}
	}

	// Format progress message, then render and print
	progressMsg := fmt.Sprintf("Deleting %d clusters...", len(clusters))
	fmt.Println(s.Progress.Render(progressMsg))

	// Track results
	var successCount, failCount int
	var errors []string

	// Delete each cluster
	for i, cluster := range clusters {
		displayName := cluster.Alias
		if displayName == "" {
			displayName = cluster.ID
		}

		// Format progress message for each cluster
		clusterProgressMsg := fmt.Sprintf("[%d/%d] Deleting cluster '%s'...", i+1, len(clusters), displayName)
		fmt.Println(s.Progress.Render(clusterProgressMsg))

		result, err := client.API.Cluster.Delete(ctx, cluster.ID)
		if err != nil {
			failCount++
			errorMsg := fmt.Sprintf("Cluster '%s': %v", displayName, err)
			errors = append(errors, errorMsg)

			// Format error message, then render and print
			failMsg := fmt.Sprintf("Failed: %s", err)
			fmt.Println(s.Error.Render(failMsg))
			continue
		}

		// Check for partial failures using util file
		if errorSummary := utils.GetClusterDeleteErrorSummary(result); errorSummary != "" {
			failCount++
			errorMsg := fmt.Sprintf("Cluster '%s' partially failed: %s", displayName, errorSummary)
			errors = append(errors, errorMsg)

			// Format partial failure message
			partialFailMsg := fmt.Sprintf("Partially failed: %s", errorSummary)
			fmt.Println(s.Warning.Render(partialFailMsg))
		} else {
			successCount++
			fmt.Println(s.Success.Render("  ✓ Deleted successfully"))
		}
	}

	// Summary
	fmt.Println()
	fmt.Println(s.Progress.Render("=== Deletion Summary ==="))

	// Format success count message
	successMsg := fmt.Sprintf("✓ Successfully deleted: %d clusters", successCount)
	fmt.Println(s.Success.Render(successMsg))

	if failCount > 0 {
		// Format failure count message
		failMsg := fmt.Sprintf("Failed to delete: %d clusters", failCount)
		fmt.Println(s.Error.Render(failMsg))

		fmt.Println()
		fmt.Println(s.Warning.Render("Error details:"))
		for _, error := range errors {
			errorDetail := fmt.Sprintf("  • %s", error)
			fmt.Println(s.Warning.Render(errorDetail))
		}
	}

	// Clean up HEAD since we deleted everything
	if successCount > 0 {
		cleanupHeadAfterDeletion()
		fmt.Println()
		fmt.Println(s.NoData.Render("HEAD cleared (all clusters deleted)"))
	}

	if failCount > 0 {
		return fmt.Errorf("some clusters failed to delete - see details above")
	}

	fmt.Println()
	fmt.Println(s.Success.Render("All clusters deleted successfully!"))
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

		// Format warning message, then render and print
		warningMsg := fmt.Sprintf("Warning: You are about to delete cluster '%s' containing %d VMs", displayName, int(cluster.VmCount))
		fmt.Println(s.Warning.Render(warningMsg))

		// Ask for confirmation
		if !askConfirmation() {
			fmt.Println(s.NoData.Render("Operation cancelled"))
			return nil
		}
	}

	// Check if this will affect HEAD before deletion
	if headWarning := checkHeadImpactSimple(target, true); headWarning != "" && !force {
		warningMsg := fmt.Sprintf("Warning: %s", headWarning)
		fmt.Println(s.Warning.Render(warningMsg))
		if !askConfirmation() {
			fmt.Println(s.NoData.Render("Operation cancelled"))
			return nil
		}
	}

	// Format deletion progress message
	deletionMsg := fmt.Sprintf("Deleting cluster '%s'...", target)
	fmt.Println(s.Progress.Render(deletionMsg))

	// Use the improved API call with detailed error handling from main branch
	result, err := client.API.Cluster.Delete(ctx, target)
	if err != nil {
		return fmt.Errorf(s.Error.Render("failed to delete cluster: %w"), err)
	}

	// Handle errors using utility
	if utils.HandleClusterDeleteErrors(result, s) {
		// Errors were printed by the utility
	} else {
		// Format success message
		successMsg := fmt.Sprintf("✓ Cluster '%s' deleted successfully", result.Data.ClusterID)
		fmt.Println(s.Success.Render(successMsg))
	}

	// Clean up HEAD if it was pointing to a VM in this cluster
	cleanupHeadAfterDeletion()

	return nil
}

func deleteVM(ctx context.Context, target string, s *styles.KillStyles) error {
	// Show confirmation for VM deletion
	if !force {
		// Format warning message
		warningMsg := fmt.Sprintf("Warning: You are about to delete VM '%s'", target)
		fmt.Println(s.Warning.Render(warningMsg))

		if !askConfirmation() {
			fmt.Println(s.NoData.Render("Operation cancelled"))
			return nil
		}
	}

	// Check if this will affect HEAD before deletion
	if headWarning := checkHeadImpactSimple(target, false); headWarning != "" && !force {
		warningMsg := fmt.Sprintf("Warning: %s", headWarning)
		fmt.Println(s.Warning.Render(warningMsg))
		if !askConfirmation() {
			fmt.Println(s.NoData.Render("Operation cancelled"))
			return nil
		}
	}

	// Format deletion message based on force flag
	var deletionMsg string
	if force {
		deletionMsg = fmt.Sprintf("Force deleting VM '%s'...", target)
	} else {
		deletionMsg = fmt.Sprintf("Deleting VM '%s'...", target)
	}
	fmt.Println(s.Progress.Render(deletionMsg))

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
		// Format success message
		successMsg := fmt.Sprintf("✓ VM '%s' deleted successfully", target)
		fmt.Println(s.Success.Render(successMsg))
	}

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
		fmt.Println("HEAD cleared (VM no longer exists)")
	}
}

func askConfirmation() bool {
	fmt.Printf("Are you sure you want to proceed? [y/N]: ")

	reader := bufio.NewReader(os.Stdin)
	input, err := reader.ReadString('\n')
	if err != nil {
		return false
	}
	input = strings.TrimSpace(input)

	return strings.EqualFold(input, "y") || strings.EqualFold(input, "yes")
}

func init() {
	rootCmd.AddCommand(killCmd)

	// Define flags for the kill command
	killCmd.Flags().BoolVarP(&force, "force", "f", false, "Force termination without confirmation")
	killCmd.Flags().BoolVarP(&isCluster, "cluster", "c", false, "Target is a cluster instead of a VM")
	killCmd.Flags().BoolVarP(&killAll, "all", "a", false, "Delete ALL clusters (use with extreme caution)")
}
