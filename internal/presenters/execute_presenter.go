package presenters

import (
	"fmt"

	"github.com/hdresearch/vers-cli/internal/app"
)

func RenderExecute(a *app.App, v ExecuteView) {
	if v.UsedHEAD {
		fmt.Printf("Using current HEAD VM: %s\n", v.HeadID)
	}
}
