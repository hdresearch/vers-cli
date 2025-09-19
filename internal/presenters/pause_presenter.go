package presenters

import (
	"fmt"
	"github.com/hdresearch/vers-cli/internal/app"
	"github.com/hdresearch/vers-cli/styles"
)

type PauseView struct{ VMName, NewState string }

func RenderPause(a *app.App, v PauseView) {
	s := styles.NewKillStyles()
	utilsSuccess := s.Success // keep styling consistent with kill/pause
	fmt.Println(utilsSuccess.Render("SUCCESS: VM '" + v.VMName + "' paused successfully"))
	fmt.Printf(s.HeadStatus.Render("VM state: %s\n"), v.NewState)
}
