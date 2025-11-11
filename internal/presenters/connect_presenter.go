package presenters

import (
	"fmt"
	"os"

	"github.com/hdresearch/vers-cli/internal/app"
	"github.com/hdresearch/vers-cli/styles"
)

func RenderConnect(a *app.App, v ConnectView) {
	s := styles.NewStatusStyles()
    if v.UsedHEAD {
        fmt.Fprintln(a.IO.Out, s.HeadStatus.Render("Using current HEAD VM: "+v.HeadID))
    }
    fmt.Fprintf(a.IO.Out, "%s\n", s.HeadStatus.Render(fmt.Sprintf("Connecting to VM %s...", v.VMName)))
    // Flush stdout to ensure message appears before SSH connection starts
    if f, ok := a.IO.Out.(*os.File); ok {
        f.Sync()
    }
}
