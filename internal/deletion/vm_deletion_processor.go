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
