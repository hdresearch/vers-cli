package deletion

import (
	"context"
	"fmt"

	pres "github.com/hdresearch/vers-cli/internal/presenters"
	"github.com/hdresearch/vers-cli/internal/prompts"
	delsvc "github.com/hdresearch/vers-cli/internal/services/deletion"
	"github.com/hdresearch/vers-cli/internal/utils"
	"github.com/hdresearch/vers-cli/styles"
	vers "github.com/hdresearch/vers-sdk-go"
)

type ClusterDeletionProcessor struct {
	client           *vers.Client
	styles           *styles.KillStyles
	ctx              context.Context
	skipConfirmation bool
	recursive        bool
	prompter         prompts.Prompter
}

func NewClusterDeletionProcessor(client *vers.Client, s *styles.KillStyles, ctx context.Context, skipConfirmation, recursive bool, prompter prompts.Prompter) *ClusterDeletionProcessor {
	return &ClusterDeletionProcessor{
		client:           client,
		styles:           s,
		ctx:              ctx,
		skipConfirmation: skipConfirmation,
		recursive:        recursive,
		prompter:         prompter,
	}
}

// DeleteMultipleClusters processes multiple cluster identifiers one at a time
func (p *ClusterDeletionProcessor) DeleteMultipleClusters(identifiers []string) error {
	// Process items one at a time
	var successCount, failCount int
	var errors []string
	var allDeletedVMIDs []string

	if len(identifiers) > 1 {
		fmt.Printf(p.styles.Progress.Render("Processing %d clusters...")+"\n", len(identifiers))
	}

	for i, identifier := range identifiers {
		// Process cluster one at a time
		clusterInfo, err := utils.ResolveClusterIdentifier(p.ctx, p.client, identifier)
		if err != nil {
			failCount++
			errorMsg := fmt.Sprintf("Cluster '%s': failed to resolve - %v", identifier, err)
			errors = append(errors, errorMsg)
			fmt.Printf(p.styles.Error.Render("FAILED to resolve cluster '%s': %s")+"\n", identifier, err.Error())
			continue
		}

		deletedVMIDs, err := p.DeleteSingleCluster(clusterInfo, i+1, len(identifiers))
		if err != nil {
			failCount++
			errorMsg := fmt.Sprintf("Cluster '%s': %v", clusterInfo.DisplayName, err)
			errors = append(errors, errorMsg)
		} else {
			successCount++
			allDeletedVMIDs = append(allDeletedVMIDs, deletedVMIDs...)
		}
	}

	// Print summary for multiple targets
	if len(identifiers) > 1 {
		pres.PrintDeletionSummary(pres.SummaryResults{SuccessCount: successCount, FailCount: failCount, Errors: errors, ItemType: "clusters"}, p.styles)
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

// DeleteSingleCluster deletes a single cluster with pre-resolved info
// Returns the list of deleted VM IDs and any error
func (p *ClusterDeletionProcessor) DeleteSingleCluster(clusterInfo *utils.ClusterInfo, currentIndex, totalCount int) ([]string, error) {
	// Get confirmation if not skipping confirmations
	if !p.skipConfirmation {
		// deletion warning
		fmt.Println(p.styles.Warning.Render(fmt.Sprintf("Warning: You are about to delete cluster '%s' containing %d VMs", clusterInfo.DisplayName, clusterInfo.VmCount)))
		ok, _ := p.prompter.YesNo("Proceed")
		if !ok {
			pres.OperationCancelled(p.styles)
			return nil, fmt.Errorf("operation cancelled by user")
		}
		// HEAD impact
		if utils.CheckClusterImpactsHead(p.ctx, p.client, clusterInfo.ID) {
			fmt.Println(p.styles.Warning.Render("Warning: This will affect the current HEAD"))
			ok, _ = p.prompter.YesNo("Proceed")
			if !ok {
				pres.OperationCancelled(p.styles)
				return nil, fmt.Errorf("operation cancelled by user")
			}
		}
	}

	// Show progress and perform deletion
	action := p.getDeletionAction()
	return handleDeletionResultCluster(currentIndex, totalCount, action, clusterInfo.DisplayName, func() ([]string, error) { return p.deleteCluster(clusterInfo.ID) }, p.styles)
}

func (p *ClusterDeletionProcessor) DeleteAllClusters() error {
	fmt.Println(p.styles.Progress.Render("Fetching all clusters..."))

	response, err := p.client.API.Cluster.List(p.ctx)
	if err != nil {
		return fmt.Errorf(p.styles.Error.Render("failed to list clusters: %w"), err)
	}

	if len(response.Data) == 0 {
		pres.NoDataFound("No clusters found to delete.", p.styles)
		return nil
	}

	// Convert to ClusterInfo objects using data from List response
	var clusterInfos []*utils.ClusterInfo
	for _, cluster := range response.Data {
		clusterInfo := utils.CreateClusterInfoFromListResponse(cluster)
		clusterInfos = append(clusterInfos, clusterInfo)
	}

	if !p.skipConfirmation && !p.confirmDeleteAllWithInfo(clusterInfos) {
		pres.NoDataFound("Operation cancelled - input did not match 'DELETE ALL'", p.styles)
		return nil
	}

	// Process clusters one at a time
	var successCount, failCount int
	var errors []string
	var allDeletedVMIDs []string

	fmt.Printf(p.styles.Progress.Render("Processing %d clusters...")+"\n", len(clusterInfos))

	for i, clusterInfo := range clusterInfos {
		deletedVMIDs, err := p.DeleteSingleCluster(clusterInfo, i+1, len(clusterInfos))
		if err != nil {
			failCount++
			errorMsg := fmt.Sprintf("Cluster '%s': %v", clusterInfo.DisplayName, err)
			errors = append(errors, errorMsg)
		} else {
			successCount++
			allDeletedVMIDs = append(allDeletedVMIDs, deletedVMIDs...)
		}
	}

	// Print summary
	pres.PrintDeletionSummary(pres.SummaryResults{SuccessCount: successCount, FailCount: failCount, Errors: errors, ItemType: "clusters"}, p.styles)

	// Cleanup HEAD
	if len(allDeletedVMIDs) > 0 {
		if utils.CleanupAfterDeletion(allDeletedVMIDs) {
			fmt.Println(p.styles.NoData.Render("HEAD cleared (cluster VMs were deleted)"))
		}
	}

	if failCount == 0 {
		fmt.Println()
		fmt.Println(p.styles.Success.Render("All clusters processed successfully!"))
	}

	if failCount > 0 {
		return fmt.Errorf("some clusters failed to delete - see details above")
	}

	return nil
}

// confirmDeleteAllWithInfo confirms deletion of all clusters using pre-resolved cluster info
func (p *ClusterDeletionProcessor) confirmDeleteAllWithInfo(clusterInfos []*utils.ClusterInfo) bool {
	headerMsg := fmt.Sprintf("DANGER: You are about to delete ALL %d clusters and their VMs:", len(clusterInfos))
	fmt.Println(p.styles.Warning.Render(headerMsg))
	fmt.Println()

	for i, clusterInfo := range clusterInfos {
		listItem := fmt.Sprintf("  %d. Cluster '%s' (%d VMs)", i+1, clusterInfo.DisplayName, clusterInfo.VmCount)
		fmt.Println(p.styles.Warning.Render(listItem))
	}

	fmt.Println()
	fmt.Println(p.styles.Warning.Render("This action is IRREVERSIBLE and will delete ALL your data!"))
	fmt.Println()

	ok, _ := p.prompter.ConfirmExact("Type 'DELETE ALL' to confirm", "DELETE ALL")
	return ok
}

// getDeletionAction returns the appropriate action description based on flags
func (p *ClusterDeletionProcessor) getDeletionAction() string {
	if p.skipConfirmation && p.recursive {
		return "Force deleting cluster (recursive)"
	} else if p.skipConfirmation {
		return "Force deleting cluster"
	} else if p.recursive {
		return "Deleting cluster (recursive)"
	}
	return "Deleting cluster"
}

func (p *ClusterDeletionProcessor) deleteCluster(clusterID string) ([]string, error) {
	return delsvc.DeleteCluster(p.ctx, p.client, clusterID)
}

// local helper mirrors utils.HandleDeletionResult without utils dependency
func handleDeletionResultCluster(currentIndex, totalCount int, action, displayName string, deletionFunc func() ([]string, error), s *styles.KillStyles) ([]string, error) {
	pres.ProgressCounter(currentIndex, totalCount, action, displayName, s)
	deletedIDs, err := deletionFunc()
	if err != nil {
		failMsg := fmt.Sprintf("FAILED: %s", err.Error())
		fmt.Println(s.Error.Render(failMsg))
		return nil, err
	}
	pres.SuccessMessage("Deleted successfully", s)
	return deletedIDs, nil
}
