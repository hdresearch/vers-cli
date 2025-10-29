package presenters

import (
	"fmt"
	"github.com/hdresearch/vers-cli/internal/app"
)

type RunCommitView struct{ RootVmID, HeadTarget, CommitKey string }

func RenderRunCommit(a *app.App, v RunCommitView) {
	fmt.Printf("Sending request to start VM from commit %s...\n", v.CommitKey)
	fmt.Printf("VM '%s' started successfully from commit %s.\n", v.RootVmID, v.CommitKey)
	if v.HeadTarget != "" {
		fmt.Printf("HEAD now points to: %s (from commit %s)\n", v.HeadTarget, v.CommitKey)
	}
}
