package presenters

import (
	"fmt"
	"github.com/hdresearch/vers-cli/internal/app"
	"github.com/hdresearch/vers-cli/styles"
)

type RenameView struct {
	Kind  string
	ID    string
	Alias string
}

func RenderRename(a *app.App, v RenameView) {
	s := styles.NewKillStyles()
	if v.Kind == "cluster" {
		fmt.Printf(s.Success.Render("✓ Cluster '%s' renamed to '%s'\n"), v.ID, v.Alias)
		return
	}
	fmt.Printf(s.Success.Render("✓ VM '%s' renamed to '%s'\n"), v.ID, v.Alias)
}
