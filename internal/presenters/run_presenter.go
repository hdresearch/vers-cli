package presenters

import (
	"fmt"
	"github.com/hdresearch/vers-cli/internal/app"
	"github.com/hdresearch/vers-cli/styles"
)

type RunView struct{ RootVmID, VmAlias, HeadTarget string }

func RenderRun(a *app.App, v RunView) {
	fmt.Println("Sending request to start VM...")
	fmt.Printf("VM '%s' started successfully.\n", v.RootVmID)
	if v.HeadTarget != "" {
		fmt.Printf("HEAD now points to: %s\n", v.HeadTarget)
	} else {
		// Should not happen, but keep a graceful message
		fmt.Println(styles.MutedTextStyle.Render("Warning: .vers directory not found. Run 'vers init' first."))
	}
}
