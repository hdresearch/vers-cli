package deletion

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
)

type TargetType string

const (
	TargetTypeVM      TargetType = "vm"
	TargetTypeCluster TargetType = "cluster"
)

type Target struct {
	ID          string
	DisplayName string
	Type        TargetType
	VmCount     int
}

type Processor struct {
	client *vers.Client
	styles *styles.KillStyles
}

func NewProcessor(client *vers.Client, s *styles.KillStyles) *Processor {
	return &Processor{
		client: client,
		styles: s,
	}
}

func (p *Processor) DeleteTargets(ctx context.Context, targetIDs []string, targetType TargetType, force bool) error {
	targets, err := p.validateTargets(ctx, targetIDs, targetType, force)
	if err != nil {
		return err
	}

	if len(targets) > 1 {
		fmt.Printf(p.styles.Progress.Render("Processing %d %ss for deletion...\n"), len(targets), string(targetType))
	}

	// Get confirmations
	if !force {
		if !p.confirmDeletion(targets) {
			fmt.Println(p.styles.NoData.Render("Operation cancelled"))
			return nil
		}

		if !p.confirmHeadImpact(targets) {
			fmt.Println(p.styles.NoData.Render("Operation cancelled"))
			return nil
		}
	}

	// Execute deletions
	results := p.executeDeletions(ctx, targets, force)

	// Print summary and cleanup
	if len(targets) > 1 {
		p.printSummary(results, true)
	}

	if results.SuccessCount > 0 {
		p.cleanupHead()
	}

	if results.FailCount > 0 {
		return fmt.Errorf("some %ss failed to delete - see details above", targetType)
	}

	return nil
}

func (p *Processor) DeleteAllClusters(ctx context.Context, force bool) error {
	fmt.Println(p.styles.Progress.Render("Fetching all clusters..."))

	response, err := p.client.API.Cluster.List(ctx)
	if err != nil {
		return fmt.Errorf(p.styles.Error.Render("failed to list clusters: %w"), err)
	}

	if len(response.Data) == 0 {
		fmt.Println(p.styles.NoData.Render("No clusters found to delete."))
		return nil
	}

	// Convert to clusters for display
	var clusters []struct {
		DisplayName string
		VmCount     int
	}
	var targets []Target
	for _, cluster := range response.Data {
		displayName := cluster.Alias
		if displayName == "" {
			displayName = cluster.ID
		}

		clusters = append(clusters, struct {
			DisplayName string
			VmCount     int
		}{
			DisplayName: displayName,
			VmCount:     int(cluster.VmCount),
		})

		targets = append(targets, Target{
			ID:          cluster.ID,
			DisplayName: displayName,
			Type:        TargetTypeCluster,
			VmCount:     int(cluster.VmCount),
		})
	}

	// Special confirmation for delete all
	if !force {
		if !p.confirmDeleteAll(clusters) {
			fmt.Println(p.styles.NoData.Render("Operation cancelled - input did not match 'DELETE ALL'"))
			return nil
		}
	}

	// Execute deletions
	results := p.executeDeletions(ctx, targets, force)

	// Print summary and cleanup
	p.printSummary(results, true)

	if results.SuccessCount > 0 {
		p.cleanupHead()
		fmt.Println()
		fmt.Println(p.styles.NoData.Render("HEAD cleared (all clusters deleted)"))
	}

	if results.FailCount > 0 {
		return fmt.Errorf("some clusters failed to delete - see details above")
	}

	fmt.Println()
	fmt.Println(p.styles.Success.Render("All clusters deleted successfully!"))
	return nil
}

func (p *Processor) validateTargets(ctx context.Context, targetIDs []string, targetType TargetType, force bool) ([]Target, error) {
	var targets []Target

	for _, id := range targetIDs {
		target := Target{ID: id, Type: targetType}

		if targetType == TargetTypeCluster && !force {
			// Validate cluster exists and get info
			response, err := p.client.API.Cluster.Get(ctx, id)
			if err != nil {
				return nil, fmt.Errorf(p.styles.Error.Render("failed to get cluster information for '%s': %w"), id, err)
			}

			target.DisplayName = response.Data.Alias
			if target.DisplayName == "" {
				target.DisplayName = response.Data.ID
			}
			target.VmCount = int(response.Data.VmCount)
		} else {
			target.DisplayName = id
		}

		targets = append(targets, target)
	}

	return targets, nil
}

func (p *Processor) confirmDeletion(targets []Target) bool {
	if len(targets) == 1 {
		target := targets[0]
		if target.Type == TargetTypeCluster {
			fmt.Printf(p.styles.Warning.Render("Warning: You are about to delete cluster '%s' containing %d VMs\n"), target.DisplayName, target.VmCount)
		} else {
			fmt.Printf(p.styles.Warning.Render("Warning: You are about to delete VM '%s'\n"), target.DisplayName)
		}
		return p.askConfirmation()
	} else {
		// Multiple targets
		fmt.Printf(p.styles.Warning.Render("Warning: You are about to delete %d %ss:\n"), len(targets), string(targets[0].Type))
		fmt.Println()
		for i, target := range targets {
			if target.Type == TargetTypeCluster {
				listItem := fmt.Sprintf("  %d. %s (%d VMs)", i+1, target.DisplayName, target.VmCount)
				fmt.Println(p.styles.Warning.Render(listItem))
			} else {
				listItem := fmt.Sprintf("  %d. %s", i+1, target.DisplayName)
				fmt.Println(p.styles.Warning.Render(listItem))
			}
		}
		fmt.Println()
		fmt.Printf(p.styles.Warning.Render("This action is IRREVERSIBLE and will delete ALL specified %ss!\n"), string(targets[0].Type))
		return p.askConfirmation()
	}
}

func (p *Processor) confirmHeadImpact(targets []Target) bool {
	var vmIDs, clusterIDs []string

	for _, target := range targets {
		if target.Type == TargetTypeCluster {
			clusterIDs = append(clusterIDs, target.ID)
		} else {
			vmIDs = append(vmIDs, target.ID)
		}
	}

	if !p.checkBatchImpact(vmIDs, clusterIDs) {
		return true // No impact, proceed
	}

	if len(targets) == 1 {
		fmt.Println(p.styles.Warning.Render("Warning: This will affect the current HEAD"))
	} else {
		fmt.Println(p.styles.Warning.Render("Warning: Some targets will affect the current HEAD"))
	}

	return p.askConfirmation()
}

func (p *Processor) confirmDeleteAll(clusters []struct {
	DisplayName string
	VmCount     int
}) bool {
	headerMsg := fmt.Sprintf("DANGER: You are about to delete ALL %d clusters and their VMs:", len(clusters))
	fmt.Println(p.styles.Warning.Render(headerMsg))
	fmt.Println()

	// Print cluster list
	for i, cluster := range clusters {
		listItem := fmt.Sprintf("  %d. Cluster '%s' (%d VMs)", i+1, cluster.DisplayName, cluster.VmCount)
		fmt.Println(p.styles.Warning.Render(listItem))
	}

	fmt.Println()
	fmt.Println(p.styles.Warning.Render("This action is IRREVERSIBLE and will delete ALL your data!"))
	fmt.Println()

	return p.askSpecialConfirmation("DELETE ALL")
}

type DeletionResults struct {
	SuccessCount int
	FailCount    int
	Errors       []string
}

func (p *Processor) executeDeletions(ctx context.Context, targets []Target, force bool) DeletionResults {
	var results DeletionResults

	for i, target := range targets {
		action := "Deleting " + string(target.Type)
		if target.Type == TargetTypeVM && force {
			action = "Force deleting VM"
		}

		// Print progress
		if len(targets) > 1 {
			progressMsg := fmt.Sprintf("[%d/%d] %s '%s'...", i+1, len(targets), action, target.DisplayName)
			fmt.Println(p.styles.Progress.Render(progressMsg))
		} else {
			progressMsg := fmt.Sprintf("%s '%s'...", action, target.DisplayName)
			fmt.Println(p.styles.Progress.Render(progressMsg))
		}

		var err error
		if target.Type == TargetTypeCluster {
			err = p.deleteCluster(ctx, target.ID)
		} else {
			err = p.deleteVM(ctx, target.ID, force)
		}

		if err != nil {
			results.FailCount++
			errorMsg := fmt.Sprintf("%s '%s': %v", strings.Title(string(target.Type)), target.DisplayName, err)
			results.Errors = append(results.Errors, errorMsg)

			failMsg := fmt.Sprintf("  Failed: %s", err)
			fmt.Println(p.styles.Error.Render(failMsg))
		} else {
			results.SuccessCount++
			fmt.Println(p.styles.Success.Render("  ✓ Deleted successfully"))
		}
	}

	return results
}

func (p *Processor) deleteCluster(ctx context.Context, clusterID string) error {
	result, err := p.client.API.Cluster.Delete(ctx, clusterID)
	if err != nil {
		return err
	}

	if errorSummary := utils.GetClusterDeleteErrorSummary(result); errorSummary != "" {
		return fmt.Errorf("partially failed: %s", errorSummary)
	}

	return nil
}

func (p *Processor) deleteVM(ctx context.Context, vmID string, force bool) error {
	deleteParams := vers.APIVmDeleteParams{
		Recursive: vers.F(force),
	}

	result, err := p.client.API.Vm.Delete(ctx, vmID, deleteParams)
	if err != nil {
		return err
	}

	if utils.HandleVmDeleteErrors(result, p.styles) {
		return fmt.Errorf("deletion had errors")
	}

	return nil
}

func (p *Processor) printProgress(current, total int, target Target, force bool) {
	var msg string
	if total > 1 {
		if target.Type == TargetTypeVM && force {
			msg = fmt.Sprintf("[%d/%d] Force deleting VM '%s'...", current, total, target.DisplayName)
		} else {
			msg = fmt.Sprintf("[%d/%d] Deleting %s '%s'...", current, total, target.Type, target.DisplayName)
		}
	} else {
		if target.Type == TargetTypeVM && force {
			msg = fmt.Sprintf("Force deleting VM '%s'...", target.DisplayName)
		} else {
			msg = fmt.Sprintf("Deleting %s '%s'...", target.Type, target.DisplayName)
		}
	}
	fmt.Println(p.styles.Progress.Render(msg))
}

func (p *Processor) printSummary(results DeletionResults, showSummary bool) {
	if !showSummary {
		return
	}

	fmt.Println()
	fmt.Println(p.styles.Progress.Render("=== Deletion Summary ==="))

	successMsg := fmt.Sprintf("✓ Successfully deleted: %d targets", results.SuccessCount)
	fmt.Println(p.styles.Success.Render(successMsg))

	if results.FailCount > 0 {
		failMsg := fmt.Sprintf("Failed to delete: %d targets", results.FailCount)
		fmt.Println(p.styles.Error.Render(failMsg))

		fmt.Println()
		fmt.Println(p.styles.Warning.Render("Error details:"))
		for _, error := range results.Errors {
			errorDetail := fmt.Sprintf("  • %s", error)
			fmt.Println(p.styles.Warning.Render(errorDetail))
		}
	}
}

func (p *Processor) checkHeadImpact(target string, isCluster bool) bool {
	versDir := ".vers"
	headFile := filepath.Join(versDir, "HEAD")

	headData, err := os.ReadFile(headFile)
	if err != nil {
		return false
	}

	headContent := strings.TrimSpace(string(headData))
	if headContent == "" {
		return false
	}

	if isCluster {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		vmResponse, err := p.client.API.Vm.Get(ctx, headContent)
		return err == nil && vmResponse.Data.ClusterID == target
	}

	return headContent == target
}

func (p *Processor) cleanupHead() {
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

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	_, err = p.client.API.Vm.Get(ctx, headContent)
	if err != nil {
		os.WriteFile(headFile, []byte(""), 0644)
		fmt.Println("HEAD cleared (VM no longer exists)")
	}
}

func (p *Processor) checkBatchImpact(vmIDs, clusterIDs []string) bool {
	// Check if any VM or cluster affects HEAD
	for _, vmID := range vmIDs {
		if p.checkHeadImpact(vmID, false) {
			return true
		}
	}
	for _, clusterID := range clusterIDs {
		if p.checkHeadImpact(clusterID, true) {
			return true
		}
	}
	return false
}

func (p *Processor) askConfirmation() bool {
	fmt.Printf("Are you sure you want to proceed? [y/N]: ")
	var input string
	fmt.Scanln(&input)
	return strings.EqualFold(input, "y") || strings.EqualFold(input, "yes")
}

func (p *Processor) askSpecialConfirmation(requiredText string) bool {
	fmt.Printf("Type '%s' to confirm: ", requiredText)
	var input string
	fmt.Scanln(&input)
	return strings.TrimSpace(input) == requiredText
}
