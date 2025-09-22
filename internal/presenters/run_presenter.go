package presenters

import (
	"fmt"
	"github.com/hdresearch/vers-cli/internal/app"
	"github.com/hdresearch/vers-cli/styles"
)

type RunView struct{ ClusterID, RootVmID, VmAlias, HeadTarget string }

func RenderRun(a *app.App, v RunView) {
	fmt.Println("Sending request to start cluster...")
	fmt.Printf("Cluster (ID: %s) started successfully with root vm '%s'.\n", v.ClusterID, v.RootVmID)
	if v.HeadTarget != "" {
		fmt.Printf("HEAD now points to: %s\n", v.HeadTarget)
	} else {
		// Should not happen, but keep a graceful message
		fmt.Println(styles.MutedTextStyle.Render("Warning: .vers directory not found. Run 'vers init' first."))
	}
}
