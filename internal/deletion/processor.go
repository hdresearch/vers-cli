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
		// Inline ProcessingMessage
		msg := fmt.Sprintf("Processing %d %ss...", len(targets), string(targetType))
		fmt.Println(p.styles.Progress.Render(msg))
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

	// Execute deletions and handle results
	return p.executeDeletions(ctx, targets, targetType, force)
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

	// Convert to targets
	targets := make([]Target, len(response.Data))
	clusters := make([]utils.ClusterInfo, len(response.Data))

	for i, cluster := range response.Data {
		displayName := cluster.Alias
		if displayName == "" {
			displayName = cluster.ID
		}

		clusters[i] = utils.ClusterInfo{
			DisplayName: displayName,
			VmCount:     int(cluster.VmCount),
		}

		targets[i] = Target{
			ID:          cluster.ID,
			DisplayName: displayName,
			Type:        TargetTypeCluster,
			VmCount:     int(cluster.VmCount),
		}
	}

	// Special confirmation for delete all
	if !force && !p.confirmDeleteAll(clusters) {
		utils.NoDataFound("Operation cancelled - input did not match 'DELETE ALL'", p.styles)
		return nil
	}

	// Execute deletions
	err = p.executeDeletions(ctx, targets, TargetTypeCluster, force)
	if err != nil {
		return err
	}

	// Inline AllSuccessMessage
	fmt.Println()
	msg := fmt.Sprintf("All clusters processed successfully!")
	fmt.Println(p.styles.Success.Render(msg))
	return nil
}

func (p *Processor) validateTargets(ctx context.Context, targetIDs []string, targetType TargetType, force bool) ([]Target, error) {
	targets := make([]Target, len(targetIDs))

	for i, id := range targetIDs {
		target := Target{ID: id, Type: targetType, DisplayName: id}

		if targetType == TargetTypeCluster && !force {
			// Validate cluster exists and get info
			response, err := p.client.API.Cluster.Get(ctx, id)
			if err != nil {
				return nil, fmt.Errorf(p.styles.Error.Render("failed to get cluster information for '%s': %w"), id, err)
			}

			if response.Data.Alias != "" {
				target.DisplayName = response.Data.Alias
			}
			target.VmCount = int(response.Data.VmCount)
		}

		targets[i] = target
	}

	return targets, nil
}

func (p *Processor) confirmDeletion(targets []Target) bool {
	if len(targets) == 1 {
		target := targets[0]
		if target.Type == TargetTypeCluster {
			return utils.ConfirmClusterDeletion(target.DisplayName, target.VmCount, p.styles)
		}
		return utils.ConfirmDeletion("VM", target.DisplayName, p.styles)
	}

	// Multiple targets - convert to display names
	itemNames := make([]string, len(targets))
	for i, target := range targets {
		if target.Type == TargetTypeCluster {
			itemNames[i] = fmt.Sprintf("%s (%d VMs)", target.DisplayName, target.VmCount)
		} else {
			itemNames[i] = target.DisplayName
		}
	}
	return utils.ConfirmBatchDeletion(len(targets), string(targets[0].Type), itemNames, p.styles)
}

func (p *Processor) confirmHeadImpact(targets []Target) bool {
	vmIDs := make([]string, 0, len(targets))
	clusterIDs := make([]string, 0, len(targets))

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

	message := "Warning: This will affect the current HEAD"
	if len(targets) > 1 {
		message = "Warning: Some targets will affect the current HEAD"
	}
	fmt.Println(p.styles.Warning.Render(message))

	return utils.AskConfirmation()
}

func (p *Processor) confirmDeleteAll(clusters []utils.ClusterInfo) bool {
	headerMsg := fmt.Sprintf("DANGER: You are about to delete ALL %d clusters and their VMs:", len(clusters))
	fmt.Println(p.styles.Warning.Render(headerMsg))
	fmt.Println()

	// Inline PrintClusterList
	for i, cluster := range clusters {
		listItem := fmt.Sprintf("  %d. Cluster '%s' (%d VMs)", i+1, cluster.DisplayName, cluster.VmCount)
		fmt.Println(p.styles.Warning.Render(listItem))
	}

	fmt.Println()
	fmt.Println(p.styles.Warning.Render("This action is IRREVERSIBLE and will delete ALL your data!"))
	fmt.Println()

	return utils.AskSpecialConfirmation("DELETE ALL", p.styles)
}

func (p *Processor) executeDeletions(ctx context.Context, targets []Target, targetType TargetType, force bool) error {
	var successCount, failCount int
	var errors []string

	for i, target := range targets {
		action := "Deleting " + string(target.Type)
		if target.Type == TargetTypeVM && force {
			action = "Force deleting VM"
		}

		utils.ProgressCounter(i+1, len(targets), action, target.DisplayName, p.styles)

		var err error
		if target.Type == TargetTypeCluster {
			err = p.deleteCluster(ctx, target.ID)
		} else {
			err = p.deleteVM(ctx, target.ID, force)
		}

		if err != nil {
			failCount++
			errorMsg := fmt.Sprintf("%s '%s': %v", strings.Title(string(target.Type)), target.DisplayName, err)
			errors = append(errors, errorMsg)

			failMsg := fmt.Sprintf("FAILED: %s", err.Error())
			fmt.Println(p.styles.Error.Render(failMsg))
		} else {
			successCount++
			utils.SuccessMessage("Deleted successfully", p.styles)
		}
	}

	// Print summary for multiple targets
	if len(targets) > 1 {
		summaryResults := utils.SummaryResults{
			SuccessCount: successCount,
			FailCount:    failCount,
			Errors:       errors,
			ItemType:     string(targetType) + "s",
		}
		utils.PrintSummary(summaryResults, p.styles)
	}

	// Cleanup HEAD if we deleted anything
	if successCount > 0 {
		utils.CleanupAfterDeletion(ctx, p.client)
		if targetType == TargetTypeCluster && len(targets) > 1 {
			// Inline HeadClearedMessage
			fmt.Println()
			msg := fmt.Sprintf("HEAD cleared (clusters deleted)")
			fmt.Println(p.styles.NoData.Render(msg))
		}
	}

	if failCount > 0 {
		return fmt.Errorf("some %ss failed to delete - see details above", targetType)
	}

	return nil
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
