package deletion

import (
	"context"
	"fmt"
	"strings"

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
		utils.ProcessingMessage(len(targets), string(targetType)+"s", p.styles)
	}

	// Get confirmations
	if !force {
		if !p.confirmDeletion(targets) {
			utils.OperationCancelled(p.styles)
			return nil
		}

		if !p.confirmHeadImpact(targets) {
			utils.OperationCancelled(p.styles)
			return nil
		}
	}

	// Execute deletions
	results := p.executeDeletions(ctx, targets, force)

	// Print summary and cleanup
	if len(targets) > 1 {
		summaryResults := utils.SummaryResults{
			SuccessCount: results.SuccessCount,
			FailCount:    results.FailCount,
			Errors:       results.Errors,
			ItemType:     string(targetType) + "s",
		}
		utils.PrintSummary(summaryResults, p.styles)
	}

	if results.SuccessCount > 0 {
		utils.CleanupAfterDeletion(ctx, p.client)
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
		utils.NoDataFound("No clusters found to delete.", p.styles)
		return nil
	}

	// Convert to clusters for display
	var clusters []utils.ClusterInfo
	var targets []Target
	for _, cluster := range response.Data {
		displayName := cluster.Alias
		if displayName == "" {
			displayName = cluster.ID
		}

		clusters = append(clusters, utils.ClusterInfo{
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
			utils.NoDataFound("Operation cancelled - input did not match 'DELETE ALL'", p.styles)
			return nil
		}
	}

	// Execute deletions
	results := p.executeDeletions(ctx, targets, force)

	// Print summary and cleanup
	summaryResults := utils.SummaryResults{
		SuccessCount: results.SuccessCount,
		FailCount:    results.FailCount,
		Errors:       results.Errors,
		ItemType:     "clusters",
	}
	utils.PrintSummary(summaryResults, p.styles)

	if results.SuccessCount > 0 {
		utils.CleanupAfterDeletion(ctx, p.client)
		utils.HeadClearedMessage("all clusters deleted", p.styles)
	}

	if results.FailCount > 0 {
		return fmt.Errorf("some clusters failed to delete - see details above")
	}

	utils.AllSuccessMessage("clusters", p.styles)
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
			return utils.ConfirmClusterDeletion(target.DisplayName, target.VmCount, p.styles)
		} else {
			return utils.ConfirmDeletion("VM", target.DisplayName, p.styles)
		}
	} else {
		// Multiple targets - convert to string slice for batch confirmation
		var itemNames []string
		for _, target := range targets {
			if target.Type == TargetTypeCluster {
				itemNames = append(itemNames, fmt.Sprintf("%s (%d VMs)", target.DisplayName, target.VmCount))
			} else {
				itemNames = append(itemNames, target.DisplayName)
			}
		}
		return utils.ConfirmBatchDeletion(len(targets), string(targets[0].Type), itemNames, p.styles)
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

	if !utils.CheckBatchImpact(context.Background(), p.client, vmIDs, clusterIDs) {
		return true // No impact, proceed
	}

	if len(targets) == 1 {
		fmt.Println(p.styles.Warning.Render("Warning: This will affect the current HEAD"))
	} else {
		fmt.Println(p.styles.Warning.Render("Warning: Some targets will affect the current HEAD"))
	}

	return utils.AskConfirmation()
}

func (p *Processor) confirmDeleteAll(clusters []utils.ClusterInfo) bool {
	headerMsg := fmt.Sprintf("DANGER: You are about to delete ALL %d clusters and their VMs:", len(clusters))
	fmt.Println(p.styles.Warning.Render(headerMsg))
	fmt.Println()

	utils.PrintClusterList(clusters, p.styles)

	fmt.Println()
	fmt.Println(p.styles.Warning.Render("This action is IRREVERSIBLE and will delete ALL your data!"))
	fmt.Println()

	return utils.AskSpecialConfirmation("DELETE ALL", p.styles)
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

		// Use utils for progress counter
		utils.ProgressCounter(i+1, len(targets), action, target.DisplayName, p.styles)

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

			utils.ErrorMessage("Failed: "+err.Error(), p.styles)
		} else {
			results.SuccessCount++
			utils.SuccessMessage("Deleted successfully", p.styles)
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
