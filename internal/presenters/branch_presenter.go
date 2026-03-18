package presenters

import (
	"fmt"

	"github.com/hdresearch/vers-cli/internal/app"
)

func RenderBranch(a *app.App, res BranchView) {
	newIDs := res.NewIDs
	if len(newIDs) == 0 && res.NewID != "" {
		newIDs = []string{res.NewID}
	}
	if len(newIDs) == 0 {
		fmt.Println("Error: no VM IDs returned from branch operation")
		return
	}
	numNew := len(newIDs)

	if res.UsedHEAD {
		fmt.Printf("Using current HEAD VM: %s\n", res.FromID)
	}

	progressName := res.FromName
	if progressName == "" {
		progressName = res.FromID
	}
	fmt.Printf("Creating new VM from: %s\n", progressName)

	if numNew == 1 {
		fmt.Println("✓ New VM created successfully!")
		fmt.Printf("  VM ID: %s\n", newIDs[0])
		if res.NewAlias != "" {
			fmt.Printf("  Alias: %s\n", res.NewAlias)
		}
	} else {
		fmt.Printf("✓ %d new VMs created successfully!\n", numNew)
		for _, id := range newIDs {
			fmt.Printf("  - %s\n", id)
		}
	}
	fmt.Println()

	if res.CheckoutDone {
		display := res.NewAlias
		if display == "" {
			display = newIDs[0]
		}
		fmt.Printf("✓ HEAD now points to: %s\n", display)
		return
	}

	if res.CheckoutErr != nil {
		fmt.Printf("WARNING: Failed to update HEAD: %v\n", res.CheckoutErr)
		return
	}

	if numNew == 1 {
		switchTarget := res.NewAlias
		if switchTarget == "" {
			switchTarget = newIDs[0]
		}
		fmt.Println("Tip: Use --checkout or -c to switch to the new VM")
		fmt.Printf("Tip: Run 'vers checkout %s' to switch to this VM\n", switchTarget)
	} else {
		fmt.Println("Tip: Use 'vers checkout <vm-id>' to switch to any of the new VMs")
	}
}
