package presenters

import (
	"fmt"
	"github.com/hdresearch/vers-cli/internal/app"
)

type BuildView struct {
	RootfsName string
	Skipped    bool
	Reason     string
}

func RenderBuild(a *app.App, v BuildView) {
	if v.Skipped {
		fmt.Printf("Builder is set to 'none'; skipping\n")
		return
	}
	fmt.Printf("Successfully uploaded rootfs: %s\n", v.RootfsName)
}
