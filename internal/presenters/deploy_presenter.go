package presenters

import (
	"fmt"

	"github.com/hdresearch/vers-cli/internal/app"
)

// RenderDeploy prints human-friendly output for a deploy operation.
func RenderDeploy(a *app.App, v DeployView) {
	fmt.Fprintf(a.IO.Out, "Deploy initiated for project %s\n", v.ProjectID)
	fmt.Fprintf(a.IO.Out, "  VM:     %s\n", v.VmID)
	fmt.Fprintf(a.IO.Out, "  Status: %s\n", v.Status)
	fmt.Fprintf(a.IO.Out, "\nHEAD now points to: %s\n", v.VmID)
	fmt.Fprintf(a.IO.Out, "The deploy is running in the background on the VM.\n")
}
