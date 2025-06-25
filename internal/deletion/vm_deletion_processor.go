package deletion

import (
	"context"
	"fmt"

	"github.com/hdresearch/vers-cli/internal/utils"
	"github.com/hdresearch/vers-cli/styles"
	vers "github.com/hdresearch/vers-sdk-go"
)

type VMProcessor struct {
	client *vers.Client
	styles *styles.KillStyles
	ctx    context.Context
	force  bool
}

func NewVMProcessor(client *vers.Client, s *styles.KillStyles, ctx context.Context, force bool) *VMProcessor {
	return &VMProcessor{
		client: client,
		styles: s,
		ctx:    ctx,
		force:  force,
	}
}

func (p *VMProcessor) DeleteVMs(vmIDs []string) error {
	// Only validate for multiple deletions to prevent partial failures
	// Single deletions can rely on backend error handling
	if !p.force && len(vmIDs) > 1 {
		if err := utils.ValidateResourcesExist(p.ctx, p.client, vmIDs, "VM", false); err != nil {
			return err
		}
	}

	if len(vmIDs) > 1 {
		msg := fmt.Sprintf("Processing %d VMs...", len(vmIDs))
		fmt.Println(p.styles.Progress.Render(msg))
	}

	// Get confirmations
	if !p.force {
		if !p.confirmVMDeletion(vmIDs) {
			utils.OperationCancelled(p.styles)
			return nil
		}

		if !utils.ConfirmHeadImpact(p.ctx, p.client, vmIDs, nil, p.styles) {
			utils.OperationCancelled(p.styles)
			return nil
		}
	}

	return p.executeVMDeletions(vmIDs)
}

func (p *VMProcessor) confirmVMDeletion(vmIDs []string) bool {
	if len(vmIDs) == 1 {
		return utils.ConfirmDeletion("VM", vmIDs[0], p.styles)
	}

	return utils.ConfirmBatchDeletion(len(vmIDs), "vm", vmIDs, p.styles)
}

func (p *VMProcessor) executeVMDeletions(vmIDs []string) error {
	var successCount, failCount int
	var errors []string
	var allDeletedVMIDs []string

	for i, vmID := range vmIDs {
		action := "Deleting VM"
		if p.force {
			action = "Force deleting VM"
		}

		utils.ProgressCounter(i+1, len(vmIDs), action, vmID, p.styles)

		deletedIDs, err := p.deleteVM(vmID)
		if err != nil {
			failCount++
			errorMsg := fmt.Sprintf("VM '%s': %v", vmID, err)
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
	if len(vmIDs) > 1 {
		summaryResults := utils.SummaryResults{
			SuccessCount: successCount,
			FailCount:    failCount,
			Errors:       errors,
			ItemType:     "VMs",
		}
		utils.PrintDeletionSummary(summaryResults, p.styles)
	}

	// Cleanup HEAD
	if len(allDeletedVMIDs) > 0 {
		if utils.CleanupAfterDeletion(allDeletedVMIDs) {
			fmt.Println(p.styles.NoData.Render("HEAD cleared (VM was deleted)"))
		}
	}

	if failCount > 0 {
		return fmt.Errorf("some VMs failed to delete - see details above")
	}

	return nil
}

func (p *VMProcessor) deleteVM(vmID string) ([]string, error) {
	deleteParams := vers.APIVmDeleteParams{
		Recursive: vers.F(p.force),
	}

	result, err := p.client.API.Vm.Delete(p.ctx, vmID, deleteParams)
	if err != nil {
		return nil, err
	}

	if utils.HandleVmDeleteErrors(result, p.styles) {
		return nil, fmt.Errorf("deletion had errors")
	}

	return result.Data.DeletedIDs, nil
}
