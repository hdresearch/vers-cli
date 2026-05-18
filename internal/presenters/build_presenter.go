package presenters

import (
	"fmt"

	"github.com/hdresearch/vers-cli/internal/app"
)

// BuildView is the rendered result of a `vers build`.
type BuildView struct {
	CommitID     string            `json:"commit_id"`
	Tag          string            `json:"tag,omitempty"`
	BuilderVmID  string            `json:"builder_vm_id,omitempty"`
	StepCount    int               `json:"steps"`
	CachedCount  int               `json:"cached_steps"`
	Cmd          []string          `json:"cmd,omitempty"`
	Entrypoint   []string          `json:"entrypoint,omitempty"`
	ExposedPorts []string          `json:"exposed_ports,omitempty"`
	Labels       map[string]string `json:"labels,omitempty"`
}

// RenderBuild prints a human-friendly summary to the app's output.
func RenderBuild(a *app.App, v BuildView) {
	w := a.IO.Out
	fmt.Fprintf(w, "\nSuccessfully built %s\n", v.CommitID)
	if v.Tag != "" {
		fmt.Fprintf(w, "Tagged as: %s\n", v.Tag)
	}
	fmt.Fprintf(w, "Steps: %d (%d cached)\n", v.StepCount, v.CachedCount)
	if v.BuilderVmID != "" {
		fmt.Fprintf(w, "Builder VM (kept): %s\n", v.BuilderVmID)
	}
	if len(v.Entrypoint) > 0 {
		fmt.Fprintf(w, "Entrypoint: %v\n", v.Entrypoint)
	}
	if len(v.Cmd) > 0 {
		fmt.Fprintf(w, "Cmd: %v\n", v.Cmd)
	}
	if len(v.ExposedPorts) > 0 {
		fmt.Fprintf(w, "Exposed: %v\n", v.ExposedPorts)
	}
}
