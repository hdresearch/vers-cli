package presenters

import (
	"fmt"

	"github.com/hdresearch/vers-cli/internal/app"
	vers "github.com/hdresearch/vers-sdk-go"
)

func RenderTagList(_ *app.App, v TagListView) {
	if len(v.Tags) == 0 {
		fmt.Println("No tags found")
		fmt.Println("Create one with: vers tag create <name> <commit-id>")
		return
	}

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

func RenderTagInfo(_ *app.App, t *vers.TagInfo) {
	fmt.Printf("Tag:         %s\n", t.TagName)
	fmt.Printf("Tag ID:      %s\n", t.TagID)
	fmt.Printf("Commit:      %s\n", t.CommitID)
	fmt.Printf("Created:     %s\n", t.CreatedAt.Format("2006-01-02 15:04:05"))
	fmt.Printf("Updated:     %s\n", t.UpdatedAt.Format("2006-01-02 15:04:05"))
	if t.Description != "" {
		fmt.Printf("Description: %s\n", t.Description)
	}
}
