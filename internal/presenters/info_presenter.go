package presenters

import (
	"fmt"

	"github.com/hdresearch/vers-cli/internal/app"
	"github.com/hdresearch/vers-cli/styles"
)

func RenderInfo(_ *app.App, v InfoView) {
	s := styles.NewStatusStyles()
	m := v.Metadata

	if v.UsedHEAD {
		fmt.Println(s.Tip.Render("Using current HEAD VM"))
	}

	fmt.Println(s.VMName.Render("VM " + m.VmID))

	fmt.Println(s.VMData.Render(fmt.Sprintf("ID:          %s", s.VMID.Render(m.VmID))))
	fmt.Println(s.VMData.Render(fmt.Sprintf("State:       %s", string(m.State))))
	fmt.Println(s.VMData.Render(fmt.Sprintf("IP:          %s", m.IP)))
	fmt.Println(s.VMData.Render(fmt.Sprintf("Owner:       %s", m.OwnerID)))
	fmt.Println(s.VMData.Render(fmt.Sprintf("Created:     %s", m.CreatedAt.Format("2006-01-02 15:04:05 UTC"))))

	if !m.DeletedAt.IsZero() {
		fmt.Println(s.VMData.Render(fmt.Sprintf("Deleted:     %s", m.DeletedAt.Format("2006-01-02 15:04:05 UTC"))))
	}

	// Lineage info
	if m.ParentCommitID != "" {
		fmt.Println(s.VMData.Render(fmt.Sprintf("Parent commit:      %s", s.VMID.Render(m.ParentCommitID))))
	}
	if m.GrandparentVmID != "" {
		fmt.Println(s.VMData.Render(fmt.Sprintf("Grandparent VM:     %s", s.VMID.Render(m.GrandparentVmID))))
	}
}
