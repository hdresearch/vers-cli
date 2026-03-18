package presenters

import (
	"fmt"

	"github.com/hdresearch/vers-cli/internal/app"
)

func RenderCommitsList(_ *app.App, v CommitsListView) {
	if v.Public {
		fmt.Println("Public Commits")
	} else {
		fmt.Println("Your Commits")
	}

	if len(v.Commits) == 0 {
		fmt.Println("No commits found")
		return
	}

	fmt.Printf("%d commit(s)\n\n", v.Total)

	fmt.Printf("%-38s  %-20s  %-8s  %s\n", "COMMIT ID", "NAME", "PUBLIC", "CREATED")
	for _, c := range v.Commits {
		name := c.Name
		if name == "" {
			name = "-"
		}
		public := "no"
		if c.IsPublic {
			public = "yes"
		}
		fmt.Printf("%-38s  %-20s  %-8s  %s\n", c.CommitID, name, public, c.CreatedAt)
	}
}

func RenderCommitParents(_ *app.App, v CommitParentsView) {
	fmt.Printf("Commit History for %s\n", v.CommitID)

	if len(v.Parents) == 0 {
		fmt.Println("No parent commits found")
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
		fmt.Printf("%s %s\n", prefix, name)
		fmt.Printf("   ID:      %s\n", p.ID)
		fmt.Printf("   Created: %s\n", p.CreatedAt.Format("2006-01-02 15:04:05"))
		if p.Description != "" {
			fmt.Printf("   Desc:    %s\n", p.Description)
		}
	}
}
