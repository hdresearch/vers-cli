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
			// Get current VM ID from HEAD (no API call)
			headVMID, err := utils.GetCurrentHeadVM()
			if err != nil {
				return fmt.Errorf("no cluster ID provided and %w", err)
			}

			fmt.Printf("Finding cluster for current HEAD VM: %s\n", headVMID)

			// OPTIMIZED: Single API call to get all clusters with complete VM data
			response, err := client.API.Cluster.List(apiCtx)
			if err != nil {
				return fmt.Errorf("failed to list clusters: %w", err)
			}
			clusters := response.Data

			// Find the cluster containing our HEAD VM
			var foundCluster *vers.APIClusterListResponseData
			for i := range clusters {
				cluster := &clusters[i]

				// Check if it's the root VM
				if cluster.RootVmID == headVMID {
					foundCluster = cluster
					break
				}

				// Check all children in the cluster
				for _, vm := range cluster.Vms {
					if vm.ID == headVMID {
						foundCluster = cluster
						break
					}
				}

				if foundCluster != nil {
					break
				}
			}

			if foundCluster == nil {
				return fmt.Errorf("couldn't find a cluster containing VM '%s'", headVMID)
			}

			// OPTIMIZED: Build tree directly from List data - no second API call needed!
			return buildAndDisplayTree(*foundCluster, headVMID)

		} else {
			// Use provided cluster identifier - we still need to resolve this
			clusterInfo, err := utils.ResolveClusterIdentifier(apiCtx, client, args[0])
			if err != nil {
				return fmt.Errorf("failed to find cluster: %w", err)
			}

			// For explicit cluster requests, we could also optimize this by using List
			// and finding the matching cluster, but since this is less common, keeping
			// the existing behavior for now
			return buildTreeFromClusterGet(apiCtx, clusterInfo)
		}
	},
}

// buildAndDisplayTree builds and displays the tree using only List API response data
func buildAndDisplayTree(cluster vers.APIClusterListResponseData, headVMID string) error {
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

// buildTreeFromClusterGet builds tree using the original Get API call (for explicit cluster requests)
func buildTreeFromClusterGet(ctx context.Context, clusterInfo *utils.ClusterInfo) error {
	fmt.Printf("Generating tree for cluster: %s\n", clusterInfo.DisplayName)

	// Fetch cluster data using the resolved cluster ID
	response, err := client.API.Cluster.Get(ctx, clusterInfo.ID)
	if err != nil {
		return fmt.Errorf("failed to get information for cluster '%s': %w", clusterInfo.DisplayName, err)
	}
	cluster := response.Data

	// Print cluster information header
	clusterHeader := styles.HeaderStyle.Render(fmt.Sprintf("Cluster: %s (Total VMs: %d)", clusterInfo.DisplayName, cluster.VmCount))
	fmt.Println(clusterHeader)

	// Find the root VM
	if cluster.RootVmID == "" {
		return fmt.Errorf("cluster '%s' has no root VM", clusterInfo.DisplayName)
	}

	// Get HEAD VM for highlighting
	headVMID, err := utils.GetCurrentHeadVM()
	if err != nil {
		// If we can't get HEAD, just print without it
		printVMTreeFromListData(cluster.Vms, cluster.RootVmID, "", true, "")
	} else {
		printVMTreeFromListData(cluster.Vms, cluster.RootVmID, "", true, headVMID)
	}

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
	if currentVM.IPAddress != "" {
		vmInfo += fmt.Sprintf(" (%s)", styles.MutedTextStyle.Render(currentVM.IPAddress))
	}

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
