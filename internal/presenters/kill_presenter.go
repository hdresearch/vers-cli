package presenters

import (
	"fmt"
	"github.com/hdresearch/vers-cli/styles"
)

// HasChildrenGuidance returns a friendly guidance message for recursive delete.
func HasChildrenGuidance(vmID string, s *styles.KillStyles) string {
	msg := fmt.Sprintf(`Cannot delete VM - it has child VMs that would be orphaned.

This VM has child VMs. Deleting it would leave them without a parent,
which could cause data inconsistency.

To delete this VM and all its children, use the --recursive (-r) flag:
  vers kill %s -r

To see the VM tree structure, run:
  vers tree`, vmID)
	return msg
}

// RootDeleteGuidance returns a friendly message for attempts to delete a cluster's root VM.
func RootDeleteGuidance(vmID string, s *styles.KillStyles) string {
	msg := fmt.Sprintf(`Cannot delete VM because it is the cluster's root VM.

Deleting the root VM would orphan the entire cluster topology.

To remove this environment, delete the whole cluster instead:
  vers kill -c <cluster-id-or-alias>

To inspect the structure and identify the cluster, run:
  vers tree

Target VM: %s`, vmID)
	return msg
}
