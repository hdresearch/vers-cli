package presenters

import (
	"fmt"

	vers "github.com/hdresearch/vers-sdk-go"
)

func RenderVMStatus(vm *vers.Vm) {
	fmt.Printf("Getting status for VM: %s\n\n", vm.VmID)
	fmt.Printf("  VM ID:    %s\n", vm.VmID)
	fmt.Printf("  State:    %s\n", vm.State)
	fmt.Printf("  Owner:    %s\n", vm.OwnerID)
	fmt.Printf("  Created:  %s\n", vm.CreatedAt.Format("2006-01-02 15:04:05"))
	fmt.Println()
	fmt.Println("Tip: To view all VMs, run: vers status")
}

func RenderVMList(vms []vers.Vm) {
	fmt.Printf("%-38s  %-10s  %s\n", "VM ID", "STATE", "CREATED")
	for _, vm := range vms {
		fmt.Printf("%-38s  %-10s  %s\n",
			vm.VmID,
			vm.State,
			vm.CreatedAt.Format("2006-01-02 15:04:05"),
		)
	}
}
