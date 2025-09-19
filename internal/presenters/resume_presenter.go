package presenters

import (
	"fmt"
	"github.com/hdresearch/vers-cli/internal/app"
	"github.com/hdresearch/vers-cli/styles"
)

type ResumeView struct{ VMName, NewState string }

func RenderResume(a *app.App, v ResumeView) {
	s := styles.NewKillStyles()
	fmt.Println(s.Success.Render("SUCCESS: VM '" + v.VMName + "' resumed successfully"))
	fmt.Printf(s.HeadStatus.Render("VM state: %s\n"), v.NewState)
}
