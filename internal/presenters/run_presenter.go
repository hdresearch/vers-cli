package presenters

import (
	"fmt"

	"github.com/hdresearch/vers-cli/internal/app"
)

type RunView struct {
	RootVmID   string `json:"vm_id"`
	VmAlias    string `json:"alias,omitempty"`
	HeadTarget string `json:"head,omitempty"`
}

func RenderRun(a *app.App, v RunView) {
	fmt.Println("Sending request to start VM...")
	fmt.Printf("VM '%s' started successfully.\n", v.RootVmID)
	if v.HeadTarget != "" {
		fmt.Printf("HEAD now points to: %s\n", v.HeadTarget)
	} else {
		fmt.Println("Warning: .vers directory not found. Run 'vers init' first.")
	}
}
