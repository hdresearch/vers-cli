package cmd

import (
	"context"
	"fmt"
	"strings"
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

			return buildAndDisplayTree(*foundCluster, headVMID)

		} else {
			response, err := client.API.Cluster.List(apiCtx)
			if err != nil {
				return fmt.Errorf("failed to list clusters: %w", err)
			}
			clusters := response.Data

			// Find the cluster by ID or alias
			var foundCluster *vers.APIClusterListResponseData
			clusterIdentifier := args[0]

			for i := range clusters {
				cluster := &clusters[i]

				// Check if it matches by ID or alias
				if cluster.ID == clusterIdentifier || cluster.Alias == clusterIdentifier {
					foundCluster = cluster
					break
				}
			}

			if foundCluster == nil {
				return fmt.Errorf("cluster '%s' not found", clusterIdentifier)
			}

			// Get HEAD VM for highlighting
			headVMID, err := utils.GetCurrentHeadVM()
			if err != nil {
				headVMID = ""
			}

			return buildAndDisplayTree(*foundCluster, headVMID)
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

	// Build header output
	var headerOutput strings.Builder
	headerOutput.WriteString(fmt.Sprintf("Generating tree for cluster: %s\n", clusterDisplayName))

	// Print cluster information header
	clusterHeader := styles.HeaderStyle.Render(fmt.Sprintf("Cluster: %s (Total VMs: %d)", clusterDisplayName, cluster.VmCount))
	headerOutput.WriteString(clusterHeader + "\n")

	// Print header information
	fmt.Print(headerOutput.String())

	// Validate we have a root VM
	if cluster.RootVmID == "" {
		return fmt.Errorf("cluster '%s' has no root VM", clusterDisplayName)
	}

	// Build the tree structure first
	var treeOutput strings.Builder
	buildVMTreeFromListData(cluster.Vms, cluster.RootVmID, "", true, headVMID, &treeOutput)

	// Build legend output
	var legendOutput strings.Builder
	legendOutput.WriteString("\nLegend:\n")
	legendOutput.WriteString(styles.MutedTextStyle.Render("- [R] Running") + "\n")
	legendOutput.WriteString(styles.MutedTextStyle.Render("- [P] Paused") + "\n")
	legendOutput.WriteString(styles.MutedTextStyle.Render("- [S] Stopped") + "\n")
	legendOutput.WriteString(styles.HelpStyle.Render("Use 'vers status -c <id>' for VM details.") + "\n")

	// Print tree and legend together
	combinedOutput := treeOutput.String() + legendOutput.String()
	fmt.Print(combinedOutput)

	return nil
}

// buildVMTreeFromListData builds tree structure into a string builder instead of printing directly
func buildVMTreeFromListData(vms []vers.VmDto, currentVMID, prefix string, isLast bool, headVMID string, output *strings.Builder) {
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

	// Add this VM's line to the output
	output.WriteString(fmt.Sprintf("%s%s%s\n", prefix, connector, finalStyle.Render(vmInfo)))

	// Prepare the prefix for children
	childPrefix := prefix
	if isLast {
		childPrefix += "    "
	} else {
		childPrefix += "│   "
	}

	// Process children recursively
	for i, childID := range currentVM.Children {
		isLastChild := i == len(currentVM.Children)-1
		buildVMTreeFromListData(vms, childID, childPrefix, isLastChild, headVMID, output)
	}
}

func init() {
	rootCmd.AddCommand(treeCmd)
}
