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
		var clusterInfo *utils.ClusterInfo
		var err error

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

			// Get all clusters and find the one containing our VM
			response, err := client.API.Cluster.List(apiCtx)
			clusters := response.Data
			if err != nil {
				return fmt.Errorf("failed to list clusters: %w", err)
			}

			found := false
			for _, cluster := range clusters {
				// First check if it's the root VM
				if cluster.RootVmID == headVMID {
					// Create ClusterInfo for the found cluster
					clusterInfo = utils.CreateClusterInfoFromListResponse(cluster)
					found = true
					break
				}

				// Check all children in the cluster
				for _, vm := range cluster.Vms {
					if vm.ID == headVMID {
						// Create ClusterInfo for the found cluster
						clusterInfo = utils.CreateClusterInfoFromListResponse(cluster)
						found = true
						break
					}
				}

				if found {
					break
				}
			}

			if !found {
				return fmt.Errorf("couldn't find a cluster containing VM '%s'", headVMID)
			}

		} else {
			// Use provided cluster identifier (could be ID or alias)
			clusterInfo, err = utils.ResolveClusterIdentifier(apiCtx, client, args[0])
			if err != nil {
				return fmt.Errorf("failed to find cluster: %w", err)
			}
		}

		fmt.Printf("Generating tree for cluster: %s\n", clusterInfo.DisplayName)

		// Fetch cluster data using the resolved cluster ID
		response, err := client.API.Cluster.Get(apiCtx, clusterInfo.ID)
		if err != nil {
			return fmt.Errorf("failed to get information for cluster '%s': %w", clusterInfo.DisplayName, err)
		}
		cluster := response.Data

		// Print cluster information header (show display name)
		clusterHeader := styles.HeaderStyle.Render(fmt.Sprintf("Cluster: %s (Total VMs: %d)", clusterInfo.DisplayName, cluster.VmCount))
		fmt.Println(clusterHeader)

		// Find the root VM
		if cluster.RootVmID == "" {
			return fmt.Errorf("cluster '%s' has no root VM", clusterInfo.DisplayName)
		}

		// Print the tree starting from the root VM - OPTIMIZED: use HEAD ID directly
		headVMID, err := utils.GetCurrentHeadVM()
		if err != nil {
			// If we can't get HEAD, just print without it
			printVMTree(cluster.Vms, cluster.RootVmID, "", true, "")
		} else {
			printVMTree(cluster.Vms, cluster.RootVmID, "", true, headVMID)
		}

		fmt.Println("\nLegend:")
		fmt.Println(styles.MutedTextStyle.Render("- [R] Running")) // Apply style to legend
		fmt.Println(styles.MutedTextStyle.Render("- [P] Paused"))
		fmt.Println(styles.MutedTextStyle.Render("- [S] Stopped"))
		fmt.Println(styles.HelpStyle.Render("Use 'vers status -c <id>' for VM details.")) // Use help style

		return nil
	},
}

// printVMTree recursively prints a tree view of VMs
func printVMTree(vms []vers.VmDto, currentVMID, prefix string, isLast bool, headVMID string) {
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
	stateStyle := styles.MutedTextStyle // Default style for state
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

	// Build the VM info string with appropriate styling
	// Show alias if available, otherwise show ID
	displayName := currentVM.Alias
	if displayName == "" {
		displayName = currentVMID
	}

	vmInfo := fmt.Sprintf("%s %s", stateStyle.Render(stateSymbol), styles.BaseTextStyle.Render(displayName))
	if currentVM.IPAddress != "" {
		vmInfo += fmt.Sprintf(" (%s)", styles.MutedTextStyle.Render(currentVM.IPAddress))
	}

	finalStyle := styles.NormalListItemStyle // Default list item style
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
		printVMTree(vms, childID, childPrefix, isLastChild, headVMID)
	}
}

func init() {
	rootCmd.AddCommand(treeCmd)
}
