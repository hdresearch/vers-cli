package handlers

import (
	"context"
	"fmt"

	"github.com/hdresearch/vers-cli/internal/app"
	"github.com/hdresearch/vers-cli/internal/presenters"
	delsvc "github.com/hdresearch/vers-cli/internal/services/deletion"
	"github.com/hdresearch/vers-cli/internal/utils"
)

type KillReq struct {
	Targets          []string
	SkipConfirmation bool
}

func HandleKill(ctx context.Context, a *app.App, r KillReq) error {
	targets := r.Targets

	// Default to HEAD if no targets
	if len(targets) == 0 {
		headVMID, err := utils.GetCurrentHeadVM()
		if err != nil {
			return fmt.Errorf("no arguments provided and %w", err)
		}
		targets = []string{headVMID}
	}

	// Confirm if needed
	if !r.SkipConfirmation {
		for _, t := range targets {
			fmt.Printf("Warning: You are about to delete VM '%s'\n", t)
		}
		ok, _ := a.Prompter.YesNo("Proceed")
		if !ok {
			presenters.OperationCancelled()
			return fmt.Errorf("operation cancelled by user")
		}
	}

	var firstErr error
	var allDeleted []string

	for i, target := range targets {
		// Resolve alias → ID
		vmInfo, err := utils.ResolveVMIdentifier(ctx, a.Client, target)
		if err != nil {
			fmt.Fprintf(a.IO.Err, "✗ Failed to resolve '%s': %v\n", target, err)
			if firstErr == nil {
				firstErr = err
			}
			continue
		}

		presenters.ProgressCounter(i+1, len(targets), "Deleting VM", vmInfo.DisplayName)

		deletedID, err := delsvc.DeleteVM(ctx, a.Client, vmInfo.ID)
		if err != nil {
			fmt.Fprintf(a.IO.Err, "✗ Failed to delete '%s': %v\n", vmInfo.DisplayName, err)
			if firstErr == nil {
				firstErr = err
			}
			continue
		}

		presenters.SuccessMessage(fmt.Sprintf("Deleted VM '%s'", vmInfo.DisplayName))
		allDeleted = append(allDeleted, deletedID)
	}

	// Clean up HEAD if any deleted VM was HEAD
	if len(allDeleted) > 0 && utils.CleanupAfterDeletion(allDeleted) {
		fmt.Println("HEAD cleared (VM was deleted)")
	}

	if len(targets) > 1 {
		presenters.PrintDeletionSummary(presenters.SummaryResults{
			SuccessCount: len(allDeleted),
			FailCount:    len(targets) - len(allDeleted),
			ItemType:     "VMs",
		})
	}

	return firstErr
}
