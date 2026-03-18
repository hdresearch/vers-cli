package presenters

import (
	"fmt"

	"github.com/hdresearch/vers-cli/internal/app"
)

type ResumeView struct{ VMName, NewState string }

func RenderResume(a *app.App, v ResumeView) {
	fmt.Printf("✓ VM '%s' resumed successfully\n", v.VMName)
	fmt.Printf("State: %s\n", v.NewState)
}
