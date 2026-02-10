package presenters

import (
	"fmt"

	"github.com/hdresearch/vers-cli/internal/app"
	"github.com/hdresearch/vers-cli/styles"
)

func RenderTunnel(a *app.App, v TunnelView) {
	s := styles.NewStatusStyles()
	if v.UsedHEAD {
		fmt.Fprintln(a.IO.Out, s.HeadStatus.Render("Using current HEAD VM: "+v.HeadID))
	}
	fmt.Fprintf(a.IO.Out, "%s\n",
		s.HeadStatus.Render(fmt.Sprintf("Forwarding 127.0.0.1:%d → %s:%d on VM %s",
			v.LocalPort, v.RemoteHost, v.RemotePort, v.VMName)))
	fmt.Fprintln(a.IO.Out, s.HeadStatus.Render("Press Ctrl-C to stop the tunnel."))
}
