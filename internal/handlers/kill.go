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
	"github.com/hdresearch/vers-cli/styles"
)

type KillReq struct {
	Targets          []string
	SkipConfirmation bool
	Recursive        bool
	IsCluster        bool
	KillAll          bool
}

// HandleKill orchestrates deletion flows. It currently prints via existing presenters/styles
// to minimize changes in Phase 1; a later phase can return DTOs for presenters to render.
func HandleKill(ctx context.Context, a *app.App, r KillReq) error {
	s := styles.NewKillStyles()

	if r.KillAll {
		processor := deletion.NewClusterDeletionProcessor(a.Client, &s, ctx, r.SkipConfirmation, r.Recursive, a.Prompter)
		return processor.DeleteAllClusters()
	}

	if len(r.Targets) == 0 {
		headVMID, err := utils.GetCurrentHeadVM()
		if err != nil {
			return fmt.Errorf(s.NoData.Render("no arguments provided and %w"), err)
		}
		fmt.Printf(s.Progress.Render("Using current HEAD VM: %s")+"\n", headVMID)

		if !r.SkipConfirmation {
			fmt.Println(s.Warning.Render("Warning: You are about to delete VM '" + headVMID + "'"))
			ok, _ := a.Prompter.YesNo("Proceed")
			if !ok {
				presenters.OperationCancelled(&s)
				return fmt.Errorf("operation cancelled by user")
			}
			if utils.CheckVMImpactsHead(headVMID) {
				fmt.Println(s.Warning.Render("Warning: This will affect the current HEAD"))
				ok, _ = a.Prompter.YesNo("Proceed")
				if !ok {
					presenters.OperationCancelled(&s)
					return fmt.Errorf("operation cancelled by user")
				}
			}
		}

		if _, err := delsvc.DeleteVM(ctx, a.Client, headVMID, r.Recursive); err != nil {
			switch e := err.(type) {
			case *errorsx.HasChildrenError:
				fmt.Println(presenters.HasChildrenGuidance(e.VMID, &s))
			case *errorsx.IsRootError:
				fmt.Println(presenters.RootDeleteGuidance(e.VMID, &s))
			default:
				return err
			}
			return fmt.Errorf("deletion had errors")
		}
		return nil
	}

	if r.IsCluster {
		processor := deletion.NewClusterDeletionProcessor(a.Client, &s, ctx, r.SkipConfirmation, r.Recursive, a.Prompter)
		return processor.DeleteMultipleClusters(r.Targets)
	}

	if len(r.Targets) == 1 {
		vmInfo, err := utils.ResolveVMIdentifier(ctx, a.Client, r.Targets[0])
		if err != nil {
			return fmt.Errorf(s.NoData.Render("failed to find VM: %w"), err)
		}
		if !r.SkipConfirmation {
			fmt.Println(s.Warning.Render("Warning: You are about to delete VM '" + vmInfo.DisplayName + "'"))
			ok, _ := a.Prompter.YesNo("Proceed")
			if !ok {
				presenters.OperationCancelled(&s)
				return fmt.Errorf("operation cancelled by user")
			}
			if utils.CheckVMImpactsHead(vmInfo.ID) {
				fmt.Println(s.Warning.Render("Warning: This will affect the current HEAD"))
				ok, _ = a.Prompter.YesNo("Proceed")
				if !ok {
					presenters.OperationCancelled(&s)
					return fmt.Errorf("operation cancelled by user")
				}
			}
		}
		if _, err := delsvc.DeleteVM(ctx, a.Client, vmInfo.ID, r.Recursive); err != nil {
			switch e := err.(type) {
			case *errorsx.HasChildrenError:
				fmt.Println(presenters.HasChildrenGuidance(e.VMID, &s))
			case *errorsx.IsRootError:
				fmt.Println(presenters.RootDeleteGuidance(e.VMID, &s))
			default:
				return err
			}
			return fmt.Errorf("deletion had errors")
		}
		return nil
	}

	processor := deletion.NewVMDeletionProcessor(a.Client, &s, ctx, r.SkipConfirmation, r.Recursive, a.Prompter)
	return processor.DeleteMultipleVMs(r.Targets)
}
