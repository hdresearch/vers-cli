package deletion

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/hdresearch/vers-cli/internal/errorsx"
	"github.com/hdresearch/vers-cli/internal/presenters"
	"github.com/hdresearch/vers-cli/internal/prompts"
	delsvc "github.com/hdresearch/vers-cli/internal/services/deletion"
	"github.com/hdresearch/vers-cli/internal/utils"
	vers "github.com/hdresearch/vers-sdk-go"
)

type VMDeletionProcessor struct {
	client           *vers.Client
	ctx              context.Context
	skipConfirmation bool
	recursive        bool
	prompter         prompts.Prompter
}

func NewVMDeletionProcessor(client *vers.Client, ctx context.Context, skipConfirmation, recursive bool, prompter prompts.Prompter) *VMDeletionProcessor {
	return &VMDeletionProcessor{
		client:           client,
		ctx:              ctx,
		skipConfirmation: skipConfirmation,
		recursive:        recursive,
		prompter:         prompter,
	}
}

func (p *VMDeletionProcessor) DeleteHeadVM(vmID, displayName string) error {
	if !p.skipConfirmation {
		fmt.Printf("Warning: You are about to delete VM '%s'\n", displayName)
		ok, _ := p.prompter.YesNo("Proceed")
		if !ok {
			presenters.OperationCancelled()
			return nil
		}
		fmt.Println("Warning: This will clear the current HEAD")
		ok, _ = p.prompter.YesNo("Proceed")
		if !ok {
			presenters.OperationCancelled()
			return nil
		}
	}

	action := p.getDeletionAction()
	deletedVMIDs, err := handleDeletionResultVM(1, 1, action, displayName, func() ([]string, error) {
		return p.deleteVM(vmID)
	})
	if err != nil {
		return err
	}

	if len(deletedVMIDs) > 0 {
		if utils.CleanupAfterDeletion(deletedVMIDs) {
			fmt.Println("HEAD cleared (VM was deleted)")
		}
	}
	return nil
}

func (p *VMDeletionProcessor) DeleteMultipleVMs(identifiers []string) error {
	var successCount, failCount int
	var errs []string
	var allDeletedVMIDs []string

	if len(identifiers) > 1 {
		fmt.Printf("Processing %d VMs...\n", len(identifiers))
	}

	for i, identifier := range identifiers {
		vmInfo, err := utils.ResolveVMIdentifier(p.ctx, p.client, identifier)
		if err != nil {
			failCount++
			errorMsg := fmt.Sprintf("VM '%s': failed to resolve - %v", identifier, err)
			errs = append(errs, errorMsg)
			fmt.Printf("✗ Failed to resolve VM '%s': %s\n", identifier, err.Error())
			continue
		}

		deletedVMIDs, err := p.DeleteSingleVM(vmInfo, i+1, len(identifiers))
		if err != nil {
			failCount++
			errorMsg := fmt.Sprintf("VM '%s': %v", vmInfo.DisplayName, err)
			errs = append(errs, errorMsg)
		} else {
			successCount++
			allDeletedVMIDs = append(allDeletedVMIDs, deletedVMIDs...)
		}
	}

	if len(identifiers) > 1 {
		presenters.PrintDeletionSummary(presenters.SummaryResults{SuccessCount: successCount, FailCount: failCount, Errors: errs, ItemType: "VMs"})
	}

	if len(allDeletedVMIDs) > 0 {
		if utils.CleanupAfterDeletion(allDeletedVMIDs) {
			fmt.Println("HEAD cleared (VM was deleted)")
		}
	}

	if failCount > 0 {
		return fmt.Errorf("some VMs failed to delete - see details above")
	}
	return nil
}

func (p *VMDeletionProcessor) DeleteSingleVM(vmInfo *utils.VMInfo, currentIndex, totalCount int) ([]string, error) {
	if !p.skipConfirmation {
		fmt.Printf("Warning: You are about to delete VM '%s'\n", vmInfo.DisplayName)
		ok, _ := p.prompter.YesNo("Proceed")
		if !ok {
			presenters.OperationCancelled()
			return nil, fmt.Errorf("operation cancelled by user")
		}
		if utils.CheckVMImpactsHead(vmInfo.ID) {
			fmt.Println("Warning: This will affect the current HEAD")
			ok, _ = p.prompter.YesNo("Proceed")
			if !ok {
				presenters.OperationCancelled()
				return nil, fmt.Errorf("operation cancelled by user")
			}
		}
	}

	action := p.getDeletionAction()
	return handleDeletionResultVM(currentIndex, totalCount, action, vmInfo.DisplayName, func() ([]string, error) {
		return p.deleteVM(vmInfo.ID)
	})
}

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
	deleted, err := delsvc.DeleteVM(p.ctx, p.client, vmID, p.recursive)
	if err != nil {
		switch e := err.(type) {
		case *errorsx.HasChildrenError:
			return nil, errors.New(presenters.HasChildrenGuidance(e.VMID))
		case *errorsx.IsRootError:
			return nil, errors.New(presenters.RootDeleteGuidance(e.VMID))
		default:
			return nil, err
		}
	}
	return deleted, nil
}

func (p *VMDeletionProcessor) isHasChildrenError(err error) bool {
	errStr := err.Error()
	return strings.Contains(errStr, "409 Conflict") && strings.Contains(errStr, "HasChildren")
}

func (p *VMDeletionProcessor) isRootError(err error) bool {
	return strings.Contains(err.Error(), "IsRoot")
}

func handleDeletionResultVM(currentIndex, totalCount int, action, displayName string, deletionFunc func() ([]string, error)) ([]string, error) {
	presenters.ProgressCounter(currentIndex, totalCount, action, displayName)
	deletedIDs, err := deletionFunc()
	if err != nil {
		fmt.Printf("✗ FAILED: %s\n", err.Error())
		return nil, err
	}
	presenters.SuccessMessage("Deleted successfully")
	return deletedIDs, nil
}
