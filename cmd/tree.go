package cmd

import (
	"context"
	"fmt"
	"strings"
	"time"

	vers "github.com/hdresearch/vers-sdk-go"
	"github.com/spf13/cobra"
)

var treeCmd = &cobra.Command{
	Use:   "tree [cluster-id]",
	Short: "Print the tree of the cluster",
	Long:  `Print a visual tree representation of the cluster and its VMs. If no cluster ID is provided, uses the cluster from current HEAD.`,
	Args:  cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		var clusterID string

		// Initialize context and client
		baseCtx := context.Background()
		client := vers.NewClient()
		apiCtx, cancel := context.WithTimeout(baseCtx, 30*time.Second)
		defer cancel()

		// If no cluster ID provided, find the cluster from HEAD
		if len(args) == 0 {
			// Get current VM ID from HEAD
			vmID, err := getCurrentHeadVM()
			if err != nil {
				return fmt.Errorf("no cluster ID provided and %w", err)
			}

			fmt.Printf("Finding cluster for current HEAD VM: %s\n", vmID)

			// Get all clusters and find the one containing our VM
			clusters, err := client.API.Cluster.List(apiCtx)
			if err != nil {
				return fmt.Errorf("failed to list clusters: %w", err)
			}

			found := false
			for _, cluster := range *clusters {
				// First check if it's the root VM
				if cluster.RootVmID == vmID {
					clusterID = cluster.ID
					found = true
					break
				}

				// Check all children in the cluster
				for _, vm := range cluster.Children {
					if vm.ID == vmID {
						clusterID = cluster.ID
						found = true
						break
					}
				}

				if found {
					break
				}
			}

			if !found {
				return fmt.Errorf("couldn't find a cluster containing VM '%s'", vmID)
			}

			fmt.Printf("Found cluster: %s\n", clusterID)
		} else {
			clusterID = args[0]
		}

		fmt.Printf("Generating tree for cluster: %s\n", clusterID)

		// Fetch cluster data
		fmt.Println("Fetching cluster information...")
		cluster, err := client.API.Cluster.Get(apiCtx, clusterID)
		if err != nil {
			return fmt.Errorf("failed to get information for cluster '%s': %w", clusterID, err)
		}

		// Print cluster information header
		fmt.Printf("\nCluster: %s (Total VMs: %d)\n", cluster.ID, cluster.VmCount)
		fmt.Println(strings.Repeat("=", 50))

		// Find the root VM
		if cluster.RootVmID == "" {
			return fmt.Errorf("cluster '%s' has no root VM", clusterID)
		}

		// Print the tree starting from the root VM
		headVM, err := getCurrentHeadVM()
		if err != nil {
			// If we can't get HEAD, just print without it
			printVMTree(cluster.Children, cluster.RootVmID, "", true, "")
		} else {
			printVMTree(cluster.Children, cluster.RootVmID, "", true, headVM)
		}

		fmt.Println("\nLegend:")
		fmt.Println("- [R] Running")
		fmt.Println("- [P] Paused")
		fmt.Println("- [S] Stopped")

		return nil
	},
}

// printVMTree recursively prints a tree view of VMs
func printVMTree(vms []vers.Vm, currentVMID, prefix string, isLast bool, headVMID string) {
	// Find the current VM in the list
	var currentVM *vers.Vm
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

	// Format state with symbol
	stateSymbol := "[?]"
	switch currentVM.State {
	case "Running":
		stateSymbol = "[R]"
	case "Paused":
		stateSymbol = "[P]"
	case "Stopped":
		stateSymbol = "[S]"
	}

	// Get short VM ID (last 8 characters) for cleaner display
	shortID := currentVM.ID
	if len(shortID) > 12 {
		parts := strings.Split(shortID, "-")
		if len(parts) > 1 {
			shortID = parts[0] + "..." + parts[len(parts)-1][:8]
		}
	}

	// Print the VM info with IP if available
	vmInfo := fmt.Sprintf("%s %s", stateSymbol, shortID)
	if currentVM.IPAddress != "" {
		vmInfo += fmt.Sprintf(" (%s)", currentVM.IPAddress)
	}
	if currentVM.ID == headVMID {
		vmInfo += " <- HEAD"
	}
	fmt.Printf("%s%s%s\n", prefix, connector, vmInfo)

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
