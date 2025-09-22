package presenters

import (
	"fmt"

	"github.com/hdresearch/vers-cli/internal/app"
	"github.com/hdresearch/vers-cli/styles"
)

func RenderBranch(a *app.App, res BranchView) {
	s := styles.NewBranchStyles()

    if res.UsedHEAD {
        fmt.Println(s.Tip.Render("Using current HEAD VM: ") + s.VMID.Render(res.FromID))
    }

	progressName := res.FromName
	if progressName == "" {
		progressName = res.FromID
	}
	fmt.Println(s.Progress.Render("Creating new VM from: " + progressName))

    fmt.Println(s.Success.Render("✓ New VM created successfully!"))
    fmt.Println(s.ListHeader.Render("New VM details:"))
    fmt.Println(s.ListItem.Render(s.InfoLabel.Render("VM ID")+": "+s.VMID.Render(res.NewID)))
	if res.NewAlias != "" {
        fmt.Println(s.ListItem.Render(s.InfoLabel.Render("Alias")+": "+s.BranchName.Render(res.NewAlias)))
    }
    fmt.Println(s.ListItem.Render(s.InfoLabel.Render("State")+": "+s.CurrentState.Render(res.NewState)))
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
	switchTarget := res.NewAlias
	if switchTarget == "" {
		switchTarget = res.NewID
	}
    fmt.Println(s.Tip.Render("Use --checkout or -c to switch to the new VM"))
    fmt.Println(s.Tip.Render("Run 'vers checkout "+switchTarget+"' to switch to this VM"))
}
