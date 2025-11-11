package errorsx

import "fmt"

// HasChildrenError indicates a VM has child VMs and non-recursive delete is unsafe.
type HasChildrenError struct{ VMID string }

func (e *HasChildrenError) Error() string { return fmt.Sprintf("vm %s has children", e.VMID) }

// IsRootError indicates a VM is a root VM.
type IsRootError struct{ VMID string }

func (e *IsRootError) Error() string { return fmt.Sprintf("vm %s is root", e.VMID) }
