package presenters

import (
	"fmt"

	"github.com/hdresearch/vers-cli/internal/app"
	"github.com/hdresearch/vers-cli/styles"
)

func RenderConnect(a *app.App, v ConnectView) {
	s := styles.NewStatusStyles()
	if v.UsedHEAD {
		fmt.Printf(s.HeadStatus.Render("Using current HEAD VM: "+v.HeadID) + "\n")
	}
	fmt.Println(s.NoData.Render("Fetching VM information..."))
	fmt.Printf(s.HeadStatus.Render("Connecting to VM %s..."), v.VMName)
	fmt.Printf("\n")
	fmt.Printf(s.HeadStatus.Render("Connecting to %s on port %s\n"), v.SSHHost, v.SSHPort)
}
