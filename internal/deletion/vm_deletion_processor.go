package deletion

import (
	"context"
	"fmt"

	"github.com/hdresearch/vers-cli/internal/utils"
	"github.com/hdresearch/vers-cli/styles"
	vers "github.com/hdresearch/vers-sdk-go"
)

type VMDeletionProcessor struct {
	client *vers.Client
	styles *styles.KillStyles
	ctx    context.Context
	force  bool
}

func NewVMDeletionProcessor(client *vers.Client, s *styles.KillStyles, ctx context.Context, force bool) *VMDeletionProcessor {
	return &VMDeletionProcessor{
		client: client,
		styles: s,
		ctx:    ctx,
		force:  force,
	}
}

// DeleteMultipleVMs processes multiple VM identifiers one at a time
func (p *VMDeletionProcessor) DeleteMultipleVMs(identifiers []string) error {
	// Process items one at a time
	var successCount, failCount int
	var errors []string
	var allDeletedVMIDs []string

	if len(identifiers) > 1 {
		fmt.Printf(p.styles.Progress.Render("Processing %d VMs...")+"\n", len(identifiers))
	}

	for i, identifier := range identifiers {
		// Process VM one at a time
		vmInfo, err := utils.ResolveVMIdentifier(p.ctx, p.client, identifier)
		if err != nil {
			failCount++
			errorMsg := fmt.Sprintf("VM '%s': failed to resolve - %v", identifier, err)
			errors = append(errors, errorMsg)
			fmt.Printf(p.styles.Error.Render("FAILED to resolve VM '%s': %s")+"\n", identifier, err.Error())
			continue
		}

		deletedVMIDs, err := p.DeleteSingleVM(vmInfo, i+1, len(identifiers))
		if err != nil {
			failCount++
			errorMsg := fmt.Sprintf("VM '%s': %v", vmInfo.DisplayName, err)
			errors = append(errors, errorMsg)
		} else {
			successCount++
			allDeletedVMIDs = append(allDeletedVMIDs, deletedVMIDs...)
		}
	}

	// Print summary for multiple targets
	if len(identifiers) > 1 {
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

// DeleteSingleVM deletes a single VM with pre-resolved info
// Returns the list of deleted VM IDs and any error
func (p *VMDeletionProcessor) DeleteSingleVM(vmInfo *utils.VMInfo, currentIndex, totalCount int) ([]string, error) {
	// Get confirmation if not forced
	if !p.force {
		if !utils.ConfirmDeletion("VM", vmInfo.DisplayName, p.styles) {
			utils.OperationCancelled(p.styles)
			return nil, fmt.Errorf("operation cancelled by user")
		}

		// Check HEAD impact for this specific VM
		if !utils.ConfirmVMHeadImpact(vmInfo.ID, p.styles) {
			utils.OperationCancelled(p.styles)
			return nil, fmt.Errorf("operation cancelled by user")
		}
	}

	// Show progress and perform deletion
	action := "Deleting VM"
	if p.force {
		action = "Force deleting VM"
	}

	return utils.HandleDeletionResult(currentIndex, totalCount, action, vmInfo.DisplayName, func() ([]string, error) {
		return p.deleteVM(vmInfo.ID)
	}, p.styles)
}

func (p *VMDeletionProcessor) deleteVM(vmID string) ([]string, error) {
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
