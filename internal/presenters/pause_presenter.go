package presenters

import (
	"fmt"

	"github.com/hdresearch/vers-cli/internal/app"
)

type PauseView struct{ VMName, NewState string }

func RenderPause(a *app.App, v PauseView) {
	fmt.Printf("✓ VM '%s' paused successfully\n", v.VMName)
	fmt.Printf("  State: %s\n", v.NewState)
}
