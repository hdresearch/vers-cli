package presenters

import (
	"fmt"

	"github.com/hdresearch/vers-cli/internal/app"
)

func RenderCopy(a *app.App, v CopyView) {
	if v.UsedHEAD {
		fmt.Printf("Using current HEAD VM: %s\n", v.HeadID)
	}
	if v.Action == "Uploading" {
		fmt.Printf("Uploading %s to VM %s at %s\n", v.Src, v.VMName, v.Dest)
	} else {
		fmt.Printf("Downloading %s from VM %s to %s\n", v.Src, v.VMName, v.Dest)
	}
	fmt.Println("File copy completed successfully")
}
