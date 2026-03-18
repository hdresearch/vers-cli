package handlers

import (
	"context"
	"fmt"

	"github.com/hdresearch/vers-cli/internal/app"
	"github.com/hdresearch/vers-cli/internal/errorsx"
	deletion "github.com/hdresearch/vers-cli/internal/handlers/deletion"
	"github.com/hdresearch/vers-cli/internal/presenters"
	delsvc "github.com/hdresearch/vers-cli/internal/services/deletion"
	"github.com/hdresearch/vers-cli/internal/utils"
)

type KillReq struct {
	Targets          []string
	SkipConfirmation bool
	Recursive        bool
}

func HandleKill(ctx context.Context, a *app.App, r KillReq) error {
	if len(r.Targets) == 0 {
		headVMID, err := utils.GetCurrentHeadVM()
		if err != nil {
			return fmt.Errorf("no arguments provided and %w", err)
		}
		fmt.Printf("Using current HEAD VM: %s\n", headVMID)

		if !r.SkipConfirmation {
			fmt.Printf("Warning: You are about to delete VM '%s'\n", headVMID)
			ok, _ := a.Prompter.YesNo("Proceed")
			if !ok {
				presenters.OperationCancelled()
				return fmt.Errorf("operation cancelled by user")
			}
			if utils.CheckVMImpactsHead(headVMID) {
				fmt.Println("Warning: This will affect the current HEAD")
				ok, _ = a.Prompter.YesNo("Proceed")
				if !ok {
					presenters.OperationCancelled()
					return fmt.Errorf("operation cancelled by user")
				}
			}
		}

		if _, err := delsvc.DeleteVM(ctx, a.Client, headVMID, r.Recursive); err != nil {
			switch e := err.(type) {
			case *errorsx.HasChildrenError:
				fmt.Println(presenters.HasChildrenGuidance(e.VMID))
			case *errorsx.IsRootError:
				fmt.Println(presenters.RootDeleteGuidance(e.VMID))
			default:
				return err
			}
			return fmt.Errorf("deletion had errors")
		}
		return nil
	}

	if len(r.Targets) == 1 {
		vmInfo, err := utils.ResolveVMIdentifier(ctx, a.Client, r.Targets[0])
		if err != nil {
			return fmt.Errorf("failed to find VM: %w", err)
		}
		if !r.SkipConfirmation {
			fmt.Printf("Warning: You are about to delete VM '%s'\n", vmInfo.DisplayName)
			ok, _ := a.Prompter.YesNo("Proceed")
			if !ok {
				presenters.OperationCancelled()
				return fmt.Errorf("operation cancelled by user")
			}
			if utils.CheckVMImpactsHead(vmInfo.ID) {
				fmt.Println("Warning: This will affect the current HEAD")
				ok, _ = a.Prompter.YesNo("Proceed")
				if !ok {
					presenters.OperationCancelled()
					return fmt.Errorf("operation cancelled by user")
				}
			}
		}
		if _, err := delsvc.DeleteVM(ctx, a.Client, vmInfo.ID, r.Recursive); err != nil {
			switch e := err.(type) {
			case *errorsx.HasChildrenError:
				fmt.Println(presenters.HasChildrenGuidance(e.VMID))
			case *errorsx.IsRootError:
				fmt.Println(presenters.RootDeleteGuidance(e.VMID))
			default:
				return err
			}
			return fmt.Errorf("deletion had errors")
		}
		return nil
	}

	processor := deletion.NewVMDeletionProcessor(a.Client, ctx, r.SkipConfirmation, r.Recursive, a.Prompter)
	return processor.DeleteMultipleVMs(r.Targets)
}
