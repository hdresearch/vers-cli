package presenters

import (
	"fmt"

	"github.com/hdresearch/vers-cli/internal/app"
)

func RenderStatus(a *app.App, res StatusView) {
	if res.Head.Show {
		switch {
		case !res.Head.Present:
			if res.Head.Empty {
				fmt.Println("HEAD: empty (create a VM with 'vers run')")
			} else {
				fmt.Println("HEAD: not a vers repository (run 'vers init' first)")
			}
			fmt.Println()
		case res.Head.DisplayName != "":
			fmt.Printf("HEAD: %s\n\n", res.Head.DisplayName)
		default:
			fmt.Printf("HEAD: %s (unable to verify)\n\n", res.Head.ID)
		}
	}

	switch res.Mode {
	case StatusVM:
		RenderVMStatus(res.VM)
	default:
		if len(res.VMs) == 0 {
			fmt.Println("No VMs found.")
			return
		}
		RenderVMList(res.VMs)
	}
}
