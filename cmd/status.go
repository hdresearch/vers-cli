package cmd

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/lipgloss/list"
	"github.com/hdresearch/vers-cli/internal/utils"
	"github.com/hdresearch/vers-cli/styles"
	"github.com/spf13/cobra"
)

// statusCmd represents the status command
var statusCmd = &cobra.Command{
	Use:   "status [vm-id]",
	Short: "Get status of clusters or VMs",
	Long:  `Displays the status of all clusters by default. Use -c flag for specific cluster details, or provide a VM ID as argument for VM-specific status.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		clusterID, _ := cmd.Flags().GetString("cluster")

		baseCtx := context.Background()
		apiCtx, cancel := context.WithTimeout(baseCtx, 30*time.Second)
		defer cancel()

		s := styles.NewStatusStyles()

		// Display current HEAD information
		displayHeadStatus()

		// If cluster flag is provided, show status for that specific cluster
		if clusterID != "" {
			fmt.Printf(s.HeadStatus.Render("Getting status for cluster: "+clusterID) + "\n")

			// Fetch cluster info
			fmt.Println(s.NoData.Render("Fetching cluster information..."))
			response, err := client.API.Cluster.Get(apiCtx, clusterID)
			if err != nil {
				return fmt.Errorf(styles.ErrorTextStyle.Render("failed to get status for cluster '%s': %w"), clusterID, err)
			}
			cluster := response.Data

			fmt.Println(s.VMListHeader.Render("Cluster details:"))
			clusterList := list.New().Enumerator(emptyEnumerator).ItemStyle(s.ClusterListItem)

			// Format cluster info
			clusterInfo := fmt.Sprintf(
				"%s\n%s\n%s",
				s.ClusterName.Render("Cluster: "+cluster.ID),
				s.ClusterData.Render("Root VM: "+s.VMID.Render(cluster.RootVmID)),
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
					vmInfo := fmt.Sprintf(
						"%s\n%s\n",
						s.ClusterData.Render("VM: "+s.VMID.Render(vm.ID)),
						s.ClusterData.Render("State: "+string(vm.State)),
					)
					vmList.Items(vmInfo)
				}
				fmt.Println(vmList)
			}

			tip := "\nTip: To view all clusters, run: vers status"
			fmt.Println(s.Tip.Render(tip))

			return nil
		}

		// If VM ID is provided as argument, show status for that specific VM
		if len(args) > 0 {
			vmID := args[0]
			fmt.Printf(s.HeadStatus.Render("Getting status for VM: "+vmID) + "\n")

			fmt.Println(s.NoData.Render("Fetching VM information..."))
			response, err := client.API.Vm.Get(apiCtx, vmID)
			if err != nil {
				return fmt.Errorf(styles.ErrorTextStyle.Render("failed to get status for VM '%s': %w"), vmID, err)
			}
			vm := response.Data

			displayHeadStatus()

			fmt.Println(s.VMListHeader.Render("VM details:"))
			vmList := list.New().Enumerator(emptyEnumerator).ItemStyle(s.ClusterListItem)

			vmInfo := fmt.Sprintf(
				"%s\n%s\n%s",
				s.ClusterName.Render("VM: "+s.VMID.Render(vm.ID)),
				s.ClusterData.Render("State: "+string(vm.State)),
				s.ClusterData.Render("Cluster: "+vm.ClusterID),
			)
			vmList.Items(vmInfo)
			fmt.Println(vmList)

			tip := "\nTip: To view the cluster containing this VM, run: vers status -c " + vm.ClusterID
			fmt.Println(s.Tip.Render(tip))

			return nil
		}

		// If no cluster ID or VM ID provided, list all clusters
		fmt.Println(s.NoData.Render("Fetching list of clusters..."))

		response, err := client.API.Cluster.List(apiCtx)
		if err != nil {
			return fmt.Errorf(styles.ErrorTextStyle.Render("failed to list clusters: %w"), err)
		}
		clusters := response.Data

		if len(clusters) == 0 {
			fmt.Println(s.NoData.Render("No clusters found."))
			return nil
		}

		fmt.Println(s.VMListHeader.Render("Available clusters:"))
		clusterList := list.New().Enumerator(emptyEnumerator).ItemStyle(s.ClusterListItem)
		for _, cluster := range clusters {
			// Combine the cluster name and its data into a single string
			clusterInfo := fmt.Sprintf(
				"%s\n%s\n%s",
				s.ClusterName.Render("Cluster: "+cluster.ID),
				s.ClusterData.Render("Root VM: "+s.VMID.Render(cluster.RootVmID)),
				s.ClusterData.Render("# children: "+fmt.Sprintf("%d", cluster.VmCount)),
			)
			clusterList.Items(clusterInfo)
		}
		fmt.Println(clusterList)

		tip := "\nTip: To view VMs in a specific cluster, use: vers status -c <cluster-id>\n" +
			"To view a specific VM, use: vers status <vm-id>"
		fmt.Println(s.Tip.Render(tip))

		return nil
	},
}

// Helper function to display current HEAD status using utils
func displayHeadStatus() error {
	s := styles.NewStatusStyles()

	headVM, err := utils.GetCurrentHeadVM()
	if err != nil {
		// Handle different error cases from utils
		errStr := err.Error()
		if strings.Contains(errStr, "HEAD not found") {
			fmt.Println(s.HeadStatus.Render("HEAD status: Not a vers repository (run 'vers init' first)"))
		} else if strings.Contains(errStr, "HEAD is empty") {
			fmt.Println(s.HeadStatus.Render("HEAD status: Empty (create a VM with 'vers run')"))
		} else {
			fmt.Printf(styles.ErrorTextStyle.Render("HEAD status: Error reading HEAD file (%v)\n"), err)
		}
		return nil
	}

	// Try to get VM details to show alias if available
	baseCtx := context.Background()
	apiCtx, cancel := context.WithTimeout(baseCtx, 5*time.Second)
	defer cancel()

	response, err := client.API.Vm.Get(apiCtx, headVM)
	if err != nil {
		fmt.Printf(s.HeadStatus.Render("HEAD status: %s (unable to verify)"), headVM)
	} else {
		vm := response.Data
		displayName := vm.Alias
		if displayName == "" {
			displayName = vm.ID
		}
		fmt.Printf(s.HeadStatus.Render("HEAD status: %s (State: %s)"), displayName, vm.State)
	}
	fmt.Println()
	return nil
}

func emptyEnumerator(_ list.Items, _ int) string {
	return ""
}

func init() {
	rootCmd.AddCommand(statusCmd)
	statusCmd.Flags().StringP("cluster", "c", "", "Cluster ID to show detailed status for")
}
