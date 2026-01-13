package presenters

import (
	"fmt"

	"github.com/hdresearch/vers-cli/internal/app"
	"github.com/hdresearch/vers-cli/styles"
)

func RenderBranch(a *app.App, res BranchView) {
	s := styles.NewBranchStyles()
	newIDs := res.NewIDs
	if len(newIDs) == 0 && res.NewID != "" {
		newIDs = []string{res.NewID}
	}
	if len(newIDs) == 0 {
		fmt.Println(s.Error.Render("No VM IDs returned from branch operation"))
		return
	}
	numNew := len(newIDs)

	if res.UsedHEAD {
		fmt.Println(s.Tip.Render("Using current HEAD VM: ") + s.VMID.Render(res.FromID))
	}

	progressName := res.FromName
	if progressName == "" {
		progressName = res.FromID
	}
	fmt.Println(s.Progress.Render("Creating new VM from: " + progressName))

	if numNew == 1 {
		fmt.Println(s.Success.Render("✓ New VM created successfully!"))
	} else {
		fmt.Println(s.Success.Render(fmt.Sprintf("✓ %d new VMs created successfully!", numNew)))
	}

	if numNew == 1 {
		fmt.Println(s.ListHeader.Render("New VM details:"))
		fmt.Println(s.ListItem.Render(s.InfoLabel.Render("VM ID") + ": " + s.VMID.Render(newIDs[0])))
		if res.NewAlias != "" {
			fmt.Println(s.ListItem.Render(s.InfoLabel.Render("Alias") + ": " + s.BranchName.Render(res.NewAlias)))
		}
	} else {
		fmt.Println(s.ListHeader.Render("New VMs:"))
		itemStyle := s.ListItem.PaddingLeft(5)
		for _, id := range newIDs {
			fmt.Println(itemStyle.Render("- " + s.VMID.Render(id)))
		}
	}
	fmt.Println()

	if res.CheckoutDone {
		successStyle := s.Success.Padding(0, 0)
		display := res.NewAlias
		if display == "" {
			display = res.NewID
		}
		fmt.Println(successStyle.Render("✓ HEAD now points to: ") + s.VMID.Render(display))
		return
	}

	if res.CheckoutErr != nil {
		warningMsg := fmt.Sprintf("WARNING: Failed to update HEAD: %v", res.CheckoutErr)
		fmt.Println(s.Warning.Render(warningMsg))
		return
	}

	// Show tip about switching when checkout not requested
	if numNew == 1 {
		switchTarget := res.NewAlias
		if switchTarget == "" {
			switchTarget = newIDs[0]
		}
		fmt.Println(s.Tip.Render("Use --checkout or -c to switch to the new VM"))
		fmt.Println(s.Tip.Render("Run 'vers checkout " + switchTarget + "' to switch to this VM"))
		return
	}

	fmt.Println(s.Tip.Render("Use 'vers checkout <vm-id>' to switch to any of the new VMs"))
}
