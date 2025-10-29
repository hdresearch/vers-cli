package presenters

import (
	"fmt"

	vers "github.com/hdresearch/vers-sdk-go"
)

// RenderTree is deprecated - cluster tree requires cluster data which no longer exists
func RenderTree(vms []vers.Vm, headVMID string) error {
	fmt.Println("Tree view is not available - cluster concept has been removed")
	fmt.Println("Use 'vers status' to view all VMs")
	return nil
}
