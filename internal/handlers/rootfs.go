package handlers

import (
	"context"
	"fmt"

	"github.com/hdresearch/vers-cli/internal/app"
	rootfssvc "github.com/hdresearch/vers-cli/internal/services/rootfs"
)

type RootfsListReq struct{}
type RootfsListView struct{ Names []string }

func HandleRootfsList(ctx context.Context, a *app.App, _ RootfsListReq) (RootfsListView, error) {
	names, err := rootfssvc.List(ctx, a.Client)
	if err != nil {
		return RootfsListView{}, err
	}
	return RootfsListView{Names: names}, nil
}

type RootfsDeleteReq struct {
	Name  string
	Force bool
}
type RootfsDeleteView struct{ Name string }

func HandleRootfsDelete(ctx context.Context, a *app.App, r RootfsDeleteReq) (RootfsDeleteView, error) {
	if !r.Force {
		if a.Prompter != nil {
			ok, _ := a.Prompter.YesNo(fmt.Sprintf("Are you sure you want to delete rootfs '%s'? This action cannot be undone.", r.Name))
			if !ok {
				return RootfsDeleteView{}, fmt.Errorf("operation cancelled")
			}
		}
	}
	name, err := rootfssvc.Delete(ctx, a.Client, r.Name)
	if err != nil {
		return RootfsDeleteView{}, err
	}
	return RootfsDeleteView{Name: name}, nil
}

// Presenters
func RenderRootfsList(a *app.App, v RootfsListView) {
	if len(v.Names) == 0 {
		fmt.Println("No rootfs images found.")
		return
	}
	fmt.Println("Available rootfs images:")
	for _, n := range v.Names {
		fmt.Printf("- %s\n", n)
	}
}

func RenderRootfsDelete(a *app.App, v RootfsDeleteView) {
	fmt.Printf("Successfully deleted rootfs '%s'\n", v.Name)
}
