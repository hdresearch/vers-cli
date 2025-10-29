package presenters

import (
	"fmt"
	"github.com/charmbracelet/lipgloss/list"
	"github.com/hdresearch/vers-cli/internal/utils"
	"github.com/hdresearch/vers-cli/styles"
	vers "github.com/hdresearch/vers-sdk-go"
)

func RenderVMStatus(s *styles.StatusStyles, vm *vers.Vm) {
	vmInfo := utils.CreateVMInfoFromVM(*vm)
    fmt.Println(s.HeadStatus.Render("Getting status for VM: "+vmInfo.DisplayName))
	fmt.Println(s.VMListHeader.Render("VM details:"))
	vmList := list.New().Enumerator(emptyEnumerator).ItemStyle(s.ClusterListItem)
	vmInfoDisplay := fmt.Sprintf("%s\n%s\n%s",
		s.ClusterName.Render("VM: "+s.VMID.Render(vmInfo.DisplayName)),
		s.ClusterData.Render("IP: "+vm.IP),
		s.ClusterData.Render("Parent: "+vm.Parent),
	)
	vmList.Items(vmInfoDisplay)
	fmt.Println(vmList)
	fmt.Println(s.Tip.Render("\nTip: To view all VMs, run: vers status"))
}

// RenderVMList renders a list of all VMs
func RenderVMList(s *styles.StatusStyles, vms []vers.Vm) {
	fmt.Println(s.VMListHeader.Render("Available VMs:"))
	vmList := list.New().Enumerator(emptyEnumerator).ItemStyle(s.ClusterListItem)
	for _, vm := range vms {
		vmInfo := fmt.Sprintf("%s\n%s\n%s",
			s.ClusterName.Render("VM: "+vm.VmID),
			s.ClusterData.Render("IP: "+vm.IP),
			s.ClusterData.Render("Parent: "+vm.Parent),
		)
		vmList.Items(vmInfo)
	}
	fmt.Println(vmList)
}

// emptyEnumerator mirrors the local helper used in commands.
func emptyEnumerator(_ list.Items, _ int) string { return "" }
