package deletion

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/hdresearch/vers-cli/internal/utils"
	"github.com/hdresearch/vers-cli/styles"
	vers "github.com/hdresearch/vers-sdk-go"
)

type VMDeletionProcessor struct {
	client           *vers.Client
	styles           *styles.KillStyles
	ctx              context.Context
	skipConfirmation bool
	recursive        bool
}

func NewVMDeletionProcessor(client *vers.Client, s *styles.KillStyles, ctx context.Context, skipConfirmation, recursive bool) *VMDeletionProcessor {
	return &VMDeletionProcessor{
		client:           client,
		styles:           s,
		ctx:              ctx,
		skipConfirmation: skipConfirmation,
		recursive:        recursive,
	}
}

// DeleteHeadVM optimized deletion for HEAD VM (no resolution needed since HEAD is always an ID)
func (p *VMDeletionProcessor) DeleteHeadVM(vmID, displayName string) error {
	// Get confirmation if not skipping confirmations
	if !p.skipConfirmation {
		if !utils.ConfirmDeletion("VM", displayName, p.styles) {
			utils.OperationCancelled(p.styles)
			return nil
		}

		// Check HEAD impact (we know it affects HEAD since this IS the HEAD VM)
		fmt.Println(p.styles.Warning.Render("Warning: This will clear the current HEAD"))
		if !utils.AskConfirmation() {
			utils.OperationCancelled(p.styles)
			return nil
		}
	}

	// Show progress and perform deletion
	action := p.getDeletionAction()

	deletedVMIDs, err := utils.HandleDeletionResult(1, 1, action, displayName, func() ([]string, error) {
		return p.deleteVM(vmID)
	}, p.styles)

	if err != nil {
		return err
	}

	// Clear HEAD since we just deleted the HEAD VM
	if len(deletedVMIDs) > 0 {
		if utils.CleanupAfterDeletion(deletedVMIDs) {
			fmt.Println(p.styles.NoData.Render("HEAD cleared (VM was deleted)"))
		}
	}

	return nil
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
	// Get confirmation if not skipping confirmations
	if !p.skipConfirmation {
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
	action := p.getDeletionAction()

	return utils.HandleDeletionResult(currentIndex, totalCount, action, vmInfo.DisplayName, func() ([]string, error) {
		return p.deleteVM(vmInfo.ID)
	}, p.styles)
}

// getDeletionAction returns the appropriate action description based on flags
func (p *VMDeletionProcessor) getDeletionAction() string {
	if p.skipConfirmation && p.recursive {
		return "Force deleting VM (recursive)"
	} else if p.skipConfirmation {
		return "Force deleting VM"
	} else if p.recursive {
		return "Deleting VM (recursive)"
	}
	return "Deleting VM"
}

func (p *VMDeletionProcessor) deleteVM(vmID string) ([]string, error) {
	deleteParams := vers.APIVmDeleteParams{
		Recursive: vers.F(p.recursive), // Use the recursive flag specifically
	}

    result, err := p.client.API.Vm.Delete(p.ctx, vmID, deleteParams)
    if err != nil {
        // Provide friendly error messages for common server-side constraints
        if p.isHasChildrenError(err) {
            return nil, p.createHasChildrenError(vmID)
        }
        if p.isRootError(err) {
            return nil, p.createRootError(vmID)
        }
        return nil, err
    }

	if utils.HandleVmDeleteErrors(result, p.styles) {
		return nil, fmt.Errorf("deletion had errors")
	}

	return result.Data.DeletedIDs, nil
}

// isHasChildrenError checks if the error is a 409 Conflict with "HasChildren"
func (p *VMDeletionProcessor) isHasChildrenError(err error) bool {
    errStr := err.Error()
    return strings.Contains(errStr, "409 Conflict") && strings.Contains(errStr, "HasChildren")
}

// createHasChildrenError creates a user-friendly error message for the HasChildren scenario
func (p *VMDeletionProcessor) createHasChildrenError(vmID string) error {
	message := fmt.Sprintf(`Cannot delete VM - it has child VMs that would be orphaned.

This VM has child VMs. Deleting it would leave them without a parent,
which could cause data inconsistency.

To delete this VM and all its children, use the --recursive (-r) flag:
  vers kill %s -r

To see the VM tree structure, run:
  vers tree`, vmID)

	return errors.New(message)
}

// isRootError checks if the error indicates the VM is a cluster root VM.
func (p *VMDeletionProcessor) isRootError(err error) bool {
    errStr := err.Error()
    // Observed as 400 Bad Request with "IsRoot" token
    return strings.Contains(errStr, "IsRoot")
}

// createRootError returns a helpful message for attempts to delete the cluster root VM directly.
func (p *VMDeletionProcessor) createRootError(vmID string) error {
    message := fmt.Sprintf(`Cannot delete VM because it is the cluster's root VM.

Deleting the root VM would orphan the entire cluster topology.

To remove this environment, delete the whole cluster instead:
  vers kill -c <cluster-id-or-alias>

To inspect the structure and identify the cluster, run:
  vers tree

Target VM: %s`, vmID)
    return errors.New(message)
}
