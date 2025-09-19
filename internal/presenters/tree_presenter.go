package presenters

import (
	"fmt"

	"github.com/hdresearch/vers-cli/styles"
	vers "github.com/hdresearch/vers-sdk-go"
)

// RenderTree prints a cluster tree using cluster payload and highlights headVMID if present.
func RenderTree(cluster vers.APIClusterGetResponseData, headVMID string) error {
	clusterDisplayName := cluster.Alias
	if clusterDisplayName == "" {
		clusterDisplayName = cluster.ID
	}

	fmt.Printf("Generating tree for cluster: %s\n", clusterDisplayName)

	clusterHeader := styles.HeaderStyle.Render(fmt.Sprintf("Cluster: %s (Total VMs: %d)", clusterDisplayName, cluster.VmCount))
	fmt.Println(clusterHeader)

	if cluster.RootVmID == "" {
		return fmt.Errorf("cluster '%s' has no root VM", clusterDisplayName)
	}

	printVMTree(cluster.Vms, cluster.RootVmID, "", true, headVMID)

	fmt.Println("\nLegend:")
	fmt.Println(styles.MutedTextStyle.Render("- [R] Running"))
	fmt.Println(styles.MutedTextStyle.Render("- [P] Paused"))
	fmt.Println(styles.MutedTextStyle.Render("- [S] Stopped"))
	fmt.Println(styles.HelpStyle.Render("Use 'vers status -c <id>' for VM details."))
	return nil
}

func printVMTree(vms []vers.VmDto, currentVMID, prefix string, isLast bool, headVMID string) {
	var currentVM *vers.VmDto
	for i := range vms {
		if vms[i].ID == currentVMID {
			currentVM = &vms[i]
			break
		}
	}
	if currentVM == nil {
		return
	}

	connector := "├── "
	if isLast {
		connector = "└── "
	}

	stateSymbol := "[?]"
	stateStyle := styles.MutedTextStyle
	switch currentVM.State {
	case "Running":
		stateSymbol = "[R]"
		stateStyle = styles.BaseTextStyle.Foreground(styles.TerminalGreen)
	case "Paused":
		stateSymbol = "[P]"
		stateStyle = styles.MutedTextStyle
	case "Stopped":
		stateSymbol = "[S]"
		stateStyle = styles.ErrorTextStyle
	}

	displayName := currentVM.Alias
	if displayName == "" {
		displayName = currentVMID
	}
	vmInfo := fmt.Sprintf("%s %s", stateStyle.Render(stateSymbol), styles.BaseTextStyle.Render(displayName))
	finalStyle := styles.NormalListItemStyle
	if currentVM.ID == headVMID {
		vmInfo += " <- HEAD"
		finalStyle = styles.SelectedListItemStyle
	}
	fmt.Printf("%s%s%s\n", prefix, connector, finalStyle.Render(vmInfo))

	childPrefix := prefix
	if isLast {
		childPrefix += "    "
	} else {
		childPrefix += "│   "
	}
	for i, childID := range currentVM.Children {
		isLastChild := i == len(currentVM.Children)-1
		printVMTree(vms, childID, childPrefix, isLastChild, headVMID)
	}
}
