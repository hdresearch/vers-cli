package presenters

import "fmt"

// HasChildrenGuidance returns a friendly guidance message for recursive delete.
func HasChildrenGuidance(vmID string) string {
	return fmt.Sprintf(`Cannot delete VM - it has child VMs that would be orphaned.

To delete this VM and all its children, use the --recursive (-r) flag:
  vers kill %s -r

To see the VM tree structure, run:
  vers tree`, vmID)
}

// RootDeleteGuidance returns a friendly message for attempts to delete a root VM.
func RootDeleteGuidance(vmID string) string {
	return fmt.Sprintf(`Cannot delete VM because it is a root VM.

Deleting the root VM would orphan the entire VM topology.

To inspect the structure, run:
  vers tree

Target VM: %s`, vmID)
}
