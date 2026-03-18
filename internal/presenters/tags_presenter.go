package presenters

import (
	"fmt"

	"github.com/hdresearch/vers-cli/internal/app"
	"github.com/hdresearch/vers-cli/styles"
	vers "github.com/hdresearch/vers-sdk-go"
)

func RenderTagList(_ *app.App, v TagListView) {
	s := styles.NewStatusStyles()

	fmt.Println(s.VMListHeader.Render("Tags"))

	if len(v.Tags) == 0 {
		fmt.Println(s.NoData.Render("No tags found"))
		fmt.Println(s.Tip.Render("Create one with: vers tag create <name> <commit-id>"))
		return
	}

	for _, t := range v.Tags {
		header := s.VMName.Render(t.TagName)
		fmt.Println(header)

		fmt.Println(s.VMData.Render(fmt.Sprintf("Commit:   %s", s.VMID.Render(t.CommitID))))
		fmt.Println(s.VMData.Render(fmt.Sprintf("Created:  %s", t.CreatedAt.Format("2006-01-02 15:04:05"))))
		if t.Description != "" {
			fmt.Println(s.VMData.Render(fmt.Sprintf("Desc:     %s", t.Description)))
		}
		fmt.Println()
	}
}

func RenderTagInfo(_ *app.App, t *vers.TagInfo) {
	s := styles.NewStatusStyles()

	fmt.Println(s.VMName.Render(t.TagName))
	fmt.Println(s.VMData.Render(fmt.Sprintf("Tag ID:      %s", t.TagID)))
	fmt.Println(s.VMData.Render(fmt.Sprintf("Commit:      %s", s.VMID.Render(t.CommitID))))
	fmt.Println(s.VMData.Render(fmt.Sprintf("Created:     %s", t.CreatedAt.Format("2006-01-02 15:04:05"))))
	fmt.Println(s.VMData.Render(fmt.Sprintf("Updated:     %s", t.UpdatedAt.Format("2006-01-02 15:04:05"))))
	if t.Description != "" {
		fmt.Println(s.VMData.Render(fmt.Sprintf("Description: %s", t.Description)))
	}
}
