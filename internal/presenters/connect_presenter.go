package presenters

import (
	"fmt"
	"os"

	"github.com/hdresearch/vers-cli/internal/app"
)

func RenderConnect(a *app.App, v ConnectView) {
	if v.UsedHEAD {
		fmt.Fprintf(a.IO.Out, "Using current HEAD VM: %s\n", v.HeadID)
	}
	fmt.Fprintf(a.IO.Out, "Connecting to VM %s...\n", v.VMName)
	if f, ok := a.IO.Out.(*os.File); ok {
		f.Sync()
	}
}
