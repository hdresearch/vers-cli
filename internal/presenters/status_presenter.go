package presenters

import (
    "fmt"
    "github.com/charmbracelet/lipgloss/list"
    "github.com/hdresearch/vers-cli/internal/utils"
    "github.com/hdresearch/vers-cli/styles"
    vers "github.com/hdresearch/vers-sdk-go"
)

func RenderClusterStatus(s *styles.StatusStyles, cluster vers.APIClusterGetResponseData) {
    var rootVMAlias string
    for _, vm := range cluster.Vms { if vm.ID == cluster.RootVmID { rootVMAlias = vm.Alias; break } }
    clusterDisplayName := cluster.Alias
    if clusterDisplayName == "" { clusterDisplayName = cluster.ID }
    rootVMDisplayName := rootVMAlias
    if rootVMDisplayName == "" { rootVMDisplayName = cluster.RootVmID }

    fmt.Printf(s.HeadStatus.Render("Getting status for cluster: "+clusterDisplayName) + "\n")
    fmt.Println(s.VMListHeader.Render("Cluster details:"))
    clusterList := list.New().Enumerator(emptyEnumerator).ItemStyle(s.ClusterListItem)
    clusterInfo := fmt.Sprintf("%s\n%s\n%s",
        s.ClusterName.Render("Cluster: "+clusterDisplayName),
        s.ClusterData.Render("Root VM: "+s.VMID.Render(rootVMDisplayName)),
        s.ClusterData.Render("# VMs: "+fmt.Sprintf("%d", len(cluster.Vms))),
    )
    clusterList.Items(clusterInfo)
    fmt.Println(clusterList)

    fmt.Println(s.VMListHeader.Render("VMs in this cluster:"))
    if len(cluster.Vms) == 0 {
        fmt.Println(s.NoData.Render("No VMs found in this cluster."))
    } else {
        vmList := list.New().Enumerator(emptyEnumerator).ItemStyle(s.ClusterListItem)
        for _, vm := range cluster.Vms {
            displayName := vm.Alias
            if displayName == "" { displayName = vm.ID }
            vmInfo := fmt.Sprintf("%s\n%s\n",
                s.ClusterData.Render("VM: "+s.VMID.Render(displayName)),
                s.ClusterData.Render("State: "+string(vm.State)),
            )
            vmList.Items(vmInfo)
        }
        fmt.Println(vmList)
    }
    fmt.Println(s.Tip.Render("\nTip: To view all clusters, run: vers status"))
}

func RenderVMStatus(s *styles.StatusStyles, vm vers.APIVmGetResponseData) {
    vmInfo := utils.CreateVMInfoFromGetResponse(vm)
    fmt.Printf(s.HeadStatus.Render("Getting status for VM: "+vmInfo.DisplayName) + "\n")
    fmt.Println(s.VMListHeader.Render("VM details:"))
    vmList := list.New().Enumerator(emptyEnumerator).ItemStyle(s.ClusterListItem)
    vmInfoDisplay := fmt.Sprintf("%s\n%s\n%s",
        s.ClusterName.Render("VM: "+s.VMID.Render(vmInfo.DisplayName)),
        s.ClusterData.Render("State: "+vmInfo.State),
        s.ClusterData.Render("Cluster: "+vm.ClusterID),
    )
    vmList.Items(vmInfoDisplay)
    fmt.Println(vmList)
    tip := "\nTip: To view the cluster containing this VM, run: vers status -c " + vm.ClusterID
    fmt.Println(s.Tip.Render(tip))
}

func RenderClusterList(s *styles.StatusStyles, clusters []vers.APIClusterListResponseData) {
    fmt.Println(s.VMListHeader.Render("Available clusters:"))
    clusterList := list.New().Enumerator(emptyEnumerator).ItemStyle(s.ClusterListItem)
    for _, cluster := range clusters {
        displayName := cluster.Alias
        if displayName == "" { displayName = cluster.ID }
        rootVMDisplayName := cluster.RootVmID
        for _, vm := range cluster.Vms {
            if vm.ID == cluster.RootVmID && vm.Alias != "" { rootVMDisplayName = vm.Alias; break }
        }
        clusterInfo := fmt.Sprintf("%s\n%s\n%s",
            s.ClusterName.Render("Cluster: "+displayName),
            s.ClusterData.Render("Root VM: "+s.VMID.Render(rootVMDisplayName)),
            s.ClusterData.Render("# children: "+fmt.Sprintf("%d", cluster.VmCount)),
        )
        clusterList.Items(clusterInfo)
    }
    fmt.Println(clusterList)
}

// emptyEnumerator mirrors the local helper used in commands.
func emptyEnumerator(_ list.Items, _ int) string { return "" }

