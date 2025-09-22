package presenters

import (
	"fmt"
	"github.com/hdresearch/vers-cli/internal/app"
)

type RunCommitView struct{ ClusterID, RootVmID, HeadTarget, CommitKey string }

func RenderRunCommit(a *app.App, v RunCommitView) {
	fmt.Printf("Sending request to start cluster from commit %s...\n", v.CommitKey)
	fmt.Printf("Cluster (ID: %s) started successfully from commit %s with root vm '%s'.\n", v.ClusterID, v.CommitKey, v.RootVmID)
	if v.HeadTarget != "" {
		fmt.Printf("HEAD now points to: %s (from commit %s)\n", v.HeadTarget, v.CommitKey)
	}
}
