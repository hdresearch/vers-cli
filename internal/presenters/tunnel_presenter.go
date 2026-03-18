package presenters

import (
	"fmt"

	"github.com/hdresearch/vers-cli/internal/app"
)

func RenderTunnel(a *app.App, v TunnelView) {
	if v.UsedHEAD {
		fmt.Fprintf(a.IO.Out, "Using current HEAD VM: %s\n", v.HeadID)
	}
	fmt.Fprintf(a.IO.Out, "Forwarding 127.0.0.1:%d → %s:%d on VM %s\n",
		v.LocalPort, v.RemoteHost, v.RemotePort, v.VMName)
	fmt.Fprintln(a.IO.Out, "Press Ctrl-C to stop the tunnel.")
}
