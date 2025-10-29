package presenters

import (
	"fmt"
	vers "github.com/hdresearch/vers-sdk-go"
)

// RenderTreeController is deprecated - tree rendering requires cluster data which no longer exists
func RenderTreeController(vms []vers.Vm, headVMID string, findingMsg string) error {
	if findingMsg != "" {
		fmt.Println(findingMsg)
	}
	fmt.Println("Tree view is not available - cluster concept has been removed")
	fmt.Println("Use 'vers status' to view all VMs")
	return nil
}
