package presenters

import (
	"fmt"

	"github.com/hdresearch/vers-cli/internal/app"
)

func RenderRepoList(_ *app.App, v RepoListView) {
	if len(v.Repositories) == 0 {
		fmt.Println("No repositories found")
		fmt.Println("Create one with: vers repo create <name>")
		return
	}

	fmt.Printf("%-24s  %-10s  %-20s  %s\n", "NAME", "VISIBILITY", "CREATED", "DESCRIPTION")
	for _, r := range v.Repositories {
		vis := "private"
		if r.IsPublic {
			vis = "public"
		}
		desc := r.Description
		if desc == "" {
			desc = "-"
		}
		fmt.Printf("%-24s  %-10s  %-20s  %s\n",
			r.Name,
			vis,
			r.CreatedAt.Format("2006-01-02 15:04:05"),
			desc,
		)
	}
}

func RenderRepoInfo(_ *app.App, r *RepoInfo) {
	vis := "private"
	if r.IsPublic {
		vis = "public"
	}
	fmt.Printf("Name:        %s\n", r.Name)
	fmt.Printf("Repo ID:     %s\n", r.RepoID)
	fmt.Printf("Visibility:  %s\n", vis)
	fmt.Printf("Created:     %s\n", r.CreatedAt.Format("2006-01-02 15:04:05"))
	if r.Description != "" {
		fmt.Printf("Description: %s\n", r.Description)
	}
}

func RenderRepoTagList(_ *app.App, v RepoTagListView) {
	if len(v.Tags) == 0 {
		fmt.Printf("No tags found in repository '%s'\n", v.Repository)
		fmt.Printf("Create one with: vers repo tag create %s <tag-name> <commit-id>\n", v.Repository)
		return
	}

	fmt.Printf("Repository: %s\n\n", v.Repository)
	fmt.Printf("%-20s  %-38s  %-20s  %s\n", "TAG", "COMMIT", "CREATED", "DESCRIPTION")
	for _, t := range v.Tags {
		desc := t.Description
		if desc == "" {
			desc = "-"
		}
		fmt.Printf("%-20s  %-38s  %-20s  %s\n",
			t.TagName,
			t.CommitID,
			t.CreatedAt.Format("2006-01-02 15:04:05"),
			desc,
		)
	}
}

func RenderRepoTagInfo(_ *app.App, t *RepoTagInfo) {
	fmt.Printf("Tag:         %s\n", t.TagName)
	fmt.Printf("Tag ID:      %s\n", t.TagID)
	fmt.Printf("Reference:   %s\n", t.Reference)
	fmt.Printf("Commit:      %s\n", t.CommitID)
	fmt.Printf("Created:     %s\n", t.CreatedAt.Format("2006-01-02 15:04:05"))
	fmt.Printf("Updated:     %s\n", t.UpdatedAt.Format("2006-01-02 15:04:05"))
	if t.Description != "" {
		fmt.Printf("Description: %s\n", t.Description)
	}
}
