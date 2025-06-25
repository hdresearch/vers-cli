package deletion

import (
	"context"
	"fmt"

	"github.com/hdresearch/vers-cli/internal/utils"
	"github.com/hdresearch/vers-cli/styles"
	vers "github.com/hdresearch/vers-sdk-go"
)

type ClusterDeletionProcessor struct {
	client *vers.Client
	styles *styles.KillStyles
	ctx    context.Context
	force  bool
}

func NewClusterDeletionProcessor(client *vers.Client, s *styles.KillStyles, ctx context.Context, force bool) *ClusterDeletionProcessor {
	return &ClusterDeletionProcessor{
		client: client,
		styles: s,
		ctx:    ctx,
		force:  force,
	}
}

func (p *ClusterDeletionProcessor) DeleteClusters(clusterIDs []string) error {
	// Only validate for multiple deletions to prevent partial failures
	// Single deletions can rely on backend error handling
	if !p.force && len(clusterIDs) > 1 {
		if err := utils.ValidateResourcesExist(p.ctx, p.client, clusterIDs, "cluster", true); err != nil {
			return err
		}
	}

	if len(clusterIDs) > 1 {
		msg := fmt.Sprintf("Processing %d clusters...", len(clusterIDs))
		fmt.Println(p.styles.Progress.Render(msg))
	}

	// Get confirmations
	if !p.force {
		if !p.confirmClusterDeletion(clusterIDs) {
			utils.OperationCancelled(p.styles)
			return nil
		}

		if !utils.ConfirmHeadImpact(p.ctx, p.client, nil, clusterIDs, p.styles) {
			utils.OperationCancelled(p.styles)
			return nil
		}
	}

	return p.executeClusterDeletions(clusterIDs)
}

func (p *ClusterDeletionProcessor) DeleteAllClusters() error {
	fmt.Println(p.styles.Progress.Render("Fetching all clusters..."))

	response, err := p.client.API.Cluster.List(p.ctx)
	if err != nil {
		return fmt.Errorf(p.styles.Error.Render("failed to list clusters: %w"), err)
	}

	if len(response.Data) == 0 {
		utils.NoDataFound("No clusters found to delete.", p.styles)
		return nil
	}

	// Extract cluster info for confirmation
	clusters := make([]utils.ClusterInfo, len(response.Data))
	clusterIDs := make([]string, len(response.Data))

	for i, cluster := range response.Data {
		displayName := cluster.Alias
		if displayName == "" {
			displayName = cluster.ID
		}

		clusters[i] = utils.ClusterInfo{
			DisplayName: displayName,
			VmCount:     int(cluster.VmCount),
		}
		clusterIDs[i] = cluster.ID
	}

	if !p.force && !p.confirmDeleteAll(clusters) {
		utils.NoDataFound("Operation cancelled - input did not match 'DELETE ALL'", p.styles)
		return nil
	}

	err = p.executeClusterDeletions(clusterIDs)
	if err != nil {
		return err
	}

	fmt.Println()
	fmt.Println(p.styles.Success.Render("All clusters processed successfully!"))
	return nil
}

func (p *ClusterDeletionProcessor) confirmClusterDeletion(clusterIDs []string) bool {
	if len(clusterIDs) == 1 {
		// Get cluster info for single cluster confirmation
		response, err := p.client.API.Cluster.Get(p.ctx, clusterIDs[0])
		if err != nil {
			// If we can't get info, fall back to simple confirmation
			return utils.ConfirmDeletion("cluster", clusterIDs[0], p.styles)
		}

		displayName := response.Data.Alias
		if displayName == "" {
			displayName = response.Data.ID
		}

		return utils.ConfirmClusterDeletion(displayName, int(response.Data.VmCount), p.styles)
	}

	// For multiple clusters, get display names with VM counts
	clusterInfos := make([]string, len(clusterIDs))
	for i, clusterID := range clusterIDs {
		response, err := p.client.API.Cluster.Get(p.ctx, clusterID)
		if err != nil {
			clusterInfos[i] = clusterID // Fallback to ID if we can't get info
		} else {
			displayName := response.Data.Alias
			if displayName == "" {
				displayName = response.Data.ID
			}
			clusterInfos[i] = fmt.Sprintf("%s (%d VMs)", displayName, response.Data.VmCount)
		}
	}

	return utils.ConfirmBatchDeletion(len(clusterIDs), "cluster", clusterInfos, p.styles)
}

func (p *ClusterDeletionProcessor) confirmDeleteAll(clusters []utils.ClusterInfo) bool {
	headerMsg := fmt.Sprintf("DANGER: You are about to delete ALL %d clusters and their VMs:", len(clusters))
	fmt.Println(p.styles.Warning.Render(headerMsg))
	fmt.Println()

	for i, cluster := range clusters {
		listItem := fmt.Sprintf("  %d. Cluster '%s' (%d VMs)", i+1, cluster.DisplayName, cluster.VmCount)
		fmt.Println(p.styles.Warning.Render(listItem))
	}

	fmt.Println()
	fmt.Println(p.styles.Warning.Render("This action is IRREVERSIBLE and will delete ALL your data!"))
	fmt.Println()

	return utils.AskSpecialConfirmation("DELETE ALL", p.styles)
}

func (p *ClusterDeletionProcessor) executeClusterDeletions(clusterIDs []string) error {
	var successCount, failCount int
	var errors []string
	var allDeletedVMIDs []string

	for i, clusterID := range clusterIDs {
		utils.ProgressCounter(i+1, len(clusterIDs), "Deleting cluster", clusterID, p.styles)

		deletedIDs, err := p.deleteCluster(clusterID)
		if err != nil {
			failCount++
			errorMsg := fmt.Sprintf("Cluster '%s': %v", clusterID, err)
			errors = append(errors, errorMsg)

			failMsg := fmt.Sprintf("FAILED: %s", err.Error())
			fmt.Println(p.styles.Error.Render(failMsg))
		} else {
			successCount++
			allDeletedVMIDs = append(allDeletedVMIDs, deletedIDs...)
			utils.SuccessMessage("Deleted successfully", p.styles)
		}
	}

	// Print summary for multiple targets
	if len(clusterIDs) > 1 {
		summaryResults := utils.SummaryResults{
			SuccessCount: successCount,
			FailCount:    failCount,
			Errors:       errors,
			ItemType:     "clusters",
		}
		utils.PrintDeletionSummary(summaryResults, p.styles)
	}

	// Cleanup HEAD
	if len(allDeletedVMIDs) > 0 {
		if utils.CleanupAfterDeletion(allDeletedVMIDs) {
			fmt.Println(p.styles.NoData.Render("HEAD cleared (cluster VMs were deleted)"))
		}
	}

	if failCount > 0 {
		return fmt.Errorf("some clusters failed to delete - see details above")
	}

	return nil
}

func (p *ClusterDeletionProcessor) deleteCluster(clusterID string) ([]string, error) {
	result, err := p.client.API.Cluster.Delete(p.ctx, clusterID)
	if err != nil {
		return nil, err
	}

	if errorSummary := utils.GetClusterDeleteErrorSummary(result); errorSummary != "" {
		return nil, fmt.Errorf("partially failed: %s", errorSummary)
	}

	return result.Data.Vms.DeletedIDs, nil
}
