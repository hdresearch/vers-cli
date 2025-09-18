package cmd

import (
	"context"
	"fmt"
	"time"

	"github.com/hdresearch/vers-cli/internal/utils"
	"github.com/hdresearch/vers-cli/styles"
	vers "github.com/hdresearch/vers-sdk-go"
	"github.com/spf13/cobra"
)

var treeCmd = &cobra.Command{
	Use:   "tree [cluster-id|cluster-alias]",
	Short: "Print the tree of the cluster",
	Long:  `Print a visual tree representation of the cluster and its VMs. If no cluster ID or alias is provided, uses the cluster from current HEAD.`,
	Args:  cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		// Initialize context
		baseCtx := context.Background()
		apiCtx, cancel := context.WithTimeout(baseCtx, 30*time.Second)
		defer cancel()

    // Resolve cluster identifier
    if len(args) == 0 {
        // Get current VM ID from HEAD
        headVMID, err := utils.GetCurrentHeadVM()
        if err != nil {
            return fmt.Errorf("no cluster ID provided and %w", err)
        }

        fmt.Printf("Finding cluster for current HEAD VM: %s\n", headVMID)

        // Get the VM to discover its cluster ID
        vmResp, err := client.API.Vm.Get(apiCtx, headVMID)
        if err != nil {
            return fmt.Errorf("failed to get HEAD VM info: %w", err)
        }

        // Fetch that cluster directly
        clResp, err := client.API.Cluster.Get(apiCtx, vmResp.Data.ClusterID)
        if err != nil {
            return fmt.Errorf("failed to get cluster '%s': %w", vmResp.Data.ClusterID, err)
        }

        return buildAndDisplayTree(clResp.Data, headVMID)

    } else {
        clusterIdentifier := args[0]

        // Fetch the cluster directly by ID or alias
        clResp, err := client.API.Cluster.Get(apiCtx, clusterIdentifier)
        if err != nil {
            return fmt.Errorf("failed to get cluster '%s': %w", clusterIdentifier, err)
        }

        // Get HEAD VM for highlighting
        headVMID, err := utils.GetCurrentHeadVM()
        if err != nil {
            headVMID = ""
        }

        return buildAndDisplayTree(clResp.Data, headVMID)
    }
	},
}

// buildAndDisplayTree builds and displays the tree using a single cluster payload
func buildAndDisplayTree(cluster vers.APIClusterGetResponseData, headVMID string) error {
	// Create display name for cluster
	clusterDisplayName := cluster.Alias
	if clusterDisplayName == "" {
		clusterDisplayName = cluster.ID
	}

	fmt.Printf("Generating tree for cluster: %s\n", clusterDisplayName)

	// Print cluster information header
	clusterHeader := styles.HeaderStyle.Render(fmt.Sprintf("Cluster: %s (Total VMs: %d)", clusterDisplayName, cluster.VmCount))
	fmt.Println(clusterHeader)

	// Validate we have a root VM
	if cluster.RootVmID == "" {
		return fmt.Errorf("cluster '%s' has no root VM", clusterDisplayName)
	}

    // Print the tree starting from the root VM
    printVMTreeFromListData(cluster.Vms, cluster.RootVmID, "", true, headVMID)

	// Print legend
	fmt.Println("\nLegend:")
	fmt.Println(styles.MutedTextStyle.Render("- [R] Running"))
	fmt.Println(styles.MutedTextStyle.Render("- [P] Paused"))
	fmt.Println(styles.MutedTextStyle.Render("- [S] Stopped"))
	fmt.Println(styles.HelpStyle.Render("Use 'vers status -c <id>' for VM details."))

	return nil
}

// printVMTreeFromListData prints tree using List response data structure
func printVMTreeFromListData(vms []vers.VmDto, currentVMID, prefix string, isLast bool, headVMID string) {
	// Find the current VM in the list
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

	// Print the current node with the correct prefix
	connector := "├── "
	if isLast {
		connector = "└── "
	}

	// Format state with symbol and style
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

	// Build the VM info string
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

	// Prepare the prefix for children
	childPrefix := prefix
	if isLast {
		childPrefix += "    "
	} else {
		childPrefix += "│   "
	}

	// Print children
	for i, childID := range currentVM.Children {
		isLastChild := i == len(currentVM.Children)-1
		printVMTreeFromListData(vms, childID, childPrefix, isLastChild, headVMID)
	}
}

func init() {
	rootCmd.AddCommand(treeCmd)
}
