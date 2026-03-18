package presenters

import (
	"fmt"

	"github.com/hdresearch/vers-cli/internal/app"
)

func RenderInfo(_ *app.App, v InfoView) {
	m := v.Metadata

	if v.UsedHEAD {
		fmt.Println("Using current HEAD VM")
	}

	fmt.Printf("VM %s\n", m.VmID)
	fmt.Printf("  State:       %s\n", string(m.State))
	fmt.Printf("  IP:          %s\n", m.IP)
	fmt.Printf("  Owner:       %s\n", m.OwnerID)
	fmt.Printf("  Created:     %s\n", m.CreatedAt.Format("2006-01-02 15:04:05 UTC"))

	if !m.DeletedAt.IsZero() {
		fmt.Printf("  Deleted:     %s\n", m.DeletedAt.Format("2006-01-02 15:04:05 UTC"))
	}

	if m.ParentCommitID != "" {
		fmt.Printf("  Parent commit:      %s\n", m.ParentCommitID)
	}
	if m.GrandparentVmID != "" {
		fmt.Printf("  Grandparent VM:     %s\n", m.GrandparentVmID)
	}
}
