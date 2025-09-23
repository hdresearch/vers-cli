package presenters

import (
	"fmt"

	"github.com/hdresearch/vers-cli/internal/app"
	"github.com/hdresearch/vers-cli/styles"
)

func RenderExecute(a *app.App, v ExecuteView) {
	s := styles.NewStatusStyles()
	if v.UsedHEAD {
		fmt.Println(s.HeadStatus.Render("Using current HEAD VM: " + v.HeadID))
	}
}
