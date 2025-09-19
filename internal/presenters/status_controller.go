package presenters

import (
	"fmt"

	"github.com/hdresearch/vers-cli/internal/app"
	"github.com/hdresearch/vers-cli/styles"
)

// RenderStatus renders the result of HandleStatus using existing status presenters.
func RenderStatus(a *app.App, res StatusView) {
	s := styles.NewStatusStyles()

	// Head line when in default mode
	if res.Head.Show {
		switch {
		case !res.Head.Present:
			if res.Head.Empty {
				fmt.Println(s.HeadStatus.Render("HEAD status: Empty (create a VM with 'vers run')"))
			} else {
				fmt.Println(s.HeadStatus.Render("HEAD status: Not a vers repository (run 'vers init' first)"))
			}
			fmt.Println()
		case res.Head.DisplayName != "":
			fmt.Printf(s.HeadStatus.Render("HEAD status: %s (State: %s)\n"), res.Head.DisplayName, res.Head.State)
			fmt.Println()
		default:
			fmt.Printf(s.HeadStatus.Render("HEAD status: %s (unable to verify)\n"), res.Head.ID)
			fmt.Println()
		}
	}

	switch res.Mode {
	case StatusCluster:
		RenderClusterStatus(&s, res.Cluster)
	case StatusVM:
		RenderVMStatus(&s, res.VM)
	default:
		if len(res.Clusters) == 0 {
			fmt.Println(s.NoData.Render("No clusters found."))
			return
		}
		RenderClusterList(&s, res.Clusters)
		tip := "\nTip: To view VMs in a specific cluster, use: vers status -c <cluster-id>\n" +
			"To view a specific VM, use: vers status <vm-id>"
		fmt.Println(s.Tip.Render(tip))
	}
}
