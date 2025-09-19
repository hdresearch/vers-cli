package presenters

import (
	"fmt"

	"github.com/hdresearch/vers-cli/internal/app"
	"github.com/hdresearch/vers-cli/styles"
)

func RenderCopy(a *app.App, v CopyView) {
	s := styles.NewStatusStyles()
	if v.UsedHEAD {
		fmt.Printf(s.HeadStatus.Render("Using current HEAD VM: "+v.HeadID) + "\n")
	}
	fmt.Println(s.NoData.Render("Fetching VM information..."))
	if v.Action == "Uploading" {
		fmt.Printf(s.HeadStatus.Render("Uploading %s to VM %s at %s\n"), v.Src, v.VMName, v.Dest)
	} else {
		fmt.Printf(s.HeadStatus.Render("Downloading %s from VM %s to %s\n"), v.Src, v.VMName, v.Dest)
	}
	fmt.Printf(s.HeadStatus.Render("File copy completed successfully\n"))
}
