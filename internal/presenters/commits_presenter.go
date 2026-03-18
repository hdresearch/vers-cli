package presenters

import (
	"fmt"

	"github.com/hdresearch/vers-cli/internal/app"
	"github.com/hdresearch/vers-cli/styles"
)

func RenderCommitsList(_ *app.App, v CommitsListView) {
	s := styles.NewStatusStyles()

	if v.Public {
		fmt.Println(s.VMListHeader.Render("Public Commits"))
	} else {
		fmt.Println(s.VMListHeader.Render("Your Commits"))
	}

	if len(v.Commits) == 0 {
		fmt.Println(s.NoData.Render("No commits found"))
		return
	}

	fmt.Printf("  %d commit(s)\n\n", v.Total)

	for _, c := range v.Commits {
		name := c.Name
		if name == "" {
			name = c.CommitID
		}
		header := s.VMName.Render(name)
		fmt.Println(header)

		fmt.Println(s.VMData.Render(fmt.Sprintf("ID:         %s", s.VMID.Render(c.CommitID))))
		fmt.Println(s.VMData.Render(fmt.Sprintf("Created:    %s", c.CreatedAt)))

		if c.IsPublic {
			fmt.Println(s.VMData.Render("Visibility: public"))
		} else {
			fmt.Println(s.VMData.Render("Visibility: private"))
		}

		if c.Description != "" {
			fmt.Println(s.VMData.Render(fmt.Sprintf("Desc:       %s", c.Description)))
		}
		if c.ParentVmID != "" {
			fmt.Println(s.VMData.Render(fmt.Sprintf("Parent VM:  %s", c.ParentVmID)))
		}
		fmt.Println()
	}
}

func RenderCommitParents(_ *app.App, v CommitParentsView) {
	s := styles.NewStatusStyles()

	fmt.Println(s.VMListHeader.Render(fmt.Sprintf("Commit History for %s", s.VMID.Render(v.CommitID))))

	if len(v.Parents) == 0 {
		fmt.Println(s.NoData.Render("No parent commits found"))
		return
	}

	for i, p := range v.Parents {
		prefix := "├─"
		if i == len(v.Parents)-1 {
			prefix = "└─"
		}
		name := p.Name
		if name == "" {
			name = p.ID
		}
		fmt.Printf("  %s %s\n", prefix, s.VMName.Render(name))
		fmt.Println(s.VMData.Render(fmt.Sprintf("     ID:      %s", s.VMID.Render(p.ID))))
		fmt.Println(s.VMData.Render(fmt.Sprintf("     Created: %s", p.CreatedAt.Format("2006-01-02 15:04:05"))))
		if p.Description != "" {
			fmt.Println(s.VMData.Render(fmt.Sprintf("     Desc:    %s", p.Description)))
		}
	}
}
