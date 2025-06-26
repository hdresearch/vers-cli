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
	Use:   "status [vm-id|alias]",
	Short: "Get status of clusters or VMs",
	Long:  `Displays the status of all clusters by default. Use -c flag for specific cluster details, or provide a VM ID or alias as argument for VM-specific status.`,
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
			// Resolve cluster identifier (could be ID or alias)
			clusterInfo, err := utils.ResolveClusterIdentifier(apiCtx, client, clusterID)
			if err != nil {
				return fmt.Errorf(styles.ErrorTextStyle.Render("failed to find cluster: %w"), err)
			}

			fmt.Printf(s.HeadStatus.Render("Getting status for cluster: "+clusterInfo.DisplayName) + "\n")

			// Fetch cluster info using resolved cluster ID
			fmt.Println(s.NoData.Render("Fetching cluster information..."))
			response, err := client.API.Cluster.Get(apiCtx, clusterInfo.ID)
			if err != nil {
				return fmt.Errorf(styles.ErrorTextStyle.Render("failed to get status for cluster '%s': %w"), clusterInfo.DisplayName, err)
			}
			cluster := response.Data

			fmt.Println(s.VMListHeader.Render("Cluster details:"))
			clusterList := list.New().Enumerator(emptyEnumerator).ItemStyle(s.ClusterListItem)

			// Format cluster info (show display name for user)
			clusterInfo_display := fmt.Sprintf(
				"%s\n%s\n%s",
				s.ClusterName.Render("Cluster: "+clusterInfo.DisplayName),
				s.ClusterData.Render("Root VM: "+s.VMID.Render(cluster.RootVmID)),
				s.ClusterData.Render("# VMs: "+fmt.Sprintf("%d", len(cluster.Vms))),
			)
			clusterList.Items(clusterInfo_display)
			fmt.Println(clusterList)

			fmt.Println(s.VMListHeader.Render("VMs in this cluster:"))

			if len(cluster.Vms) == 0 {
				fmt.Println(s.NoData.Render("No VMs found in this cluster."))
			} else {
				vmList := list.New().Enumerator(emptyEnumerator).ItemStyle(s.ClusterListItem)
				for _, vm := range cluster.Vms {
					// Show alias if available, otherwise show ID
					displayName := vm.Alias
					if displayName == "" {
						displayName = vm.ID
					}

					vmInfo := fmt.Sprintf(
						"%s\n%s\n",
						s.ClusterData.Render("VM: "+s.VMID.Render(displayName)),
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
			// Resolve VM identifier (could be ID or alias)
			vmInfo, err := utils.ResolveVMIdentifier(apiCtx, client, args[0])
			if err != nil {
				return fmt.Errorf(styles.ErrorTextStyle.Render("failed to find VM: %w"), err)
			}

			fmt.Printf(s.HeadStatus.Render("Getting status for VM: "+vmInfo.DisplayName) + "\n")

			fmt.Println(s.NoData.Render("Using VM information from resolution..."))

			displayHeadStatus()

			fmt.Println(s.VMListHeader.Render("VM details:"))
			vmList := list.New().Enumerator(emptyEnumerator).ItemStyle(s.ClusterListItem)

			// Show display name for user
			vmInfo_display := fmt.Sprintf(
				"%s\n%s\n%s",
				s.ClusterName.Render("VM: "+s.VMID.Render(vmInfo.DisplayName)),
				s.ClusterData.Render("State: "+vmInfo.State),
				s.ClusterData.Render("Cluster: (fetching...)"), // We don't have cluster ID from basic resolution
			)

			response, err := client.API.Vm.Get(apiCtx, vmInfo.ID)
			if err == nil {
				// Update display with cluster info
				vmInfo_display = fmt.Sprintf(
					"%s\n%s\n%s",
					s.ClusterName.Render("VM: "+s.VMID.Render(vmInfo.DisplayName)),
					s.ClusterData.Render("State: "+vmInfo.State),
					s.ClusterData.Render("Cluster: "+response.Data.ClusterID),
				)
			}

			vmList.Items(vmInfo_display)
			fmt.Println(vmList)

			if err == nil {
				tip := "\nTip: To view the cluster containing this VM, run: vers status -c " + response.Data.ClusterID
				fmt.Println(s.Tip.Render(tip))
			}

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
			// Show alias if available, otherwise show ID
			displayName := cluster.Alias
			if displayName == "" {
				displayName = cluster.ID
			}

			// Combine the cluster name and its data into a single string
			clusterInfo := fmt.Sprintf(
				"%s\n%s\n%s",
				s.ClusterName.Render("Cluster: "+displayName),
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

	// Get HEAD VM ID first
	headVMID, err := utils.GetCurrentHeadVM()
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
	apiCtx, cancel := context.WithTimeout(baseCtx, 3*time.Second) // Shorter timeout for status display
	defer cancel()

	response, err := client.API.Vm.Get(apiCtx, headVMID)
	if err != nil {
		fmt.Printf(s.HeadStatus.Render("HEAD status: %s (unable to verify)"), headVMID)
	} else {
		// Create VMInfo from response
		vmInfo := utils.CreateVMInfoFromGetResponse(response.Data)
		fmt.Printf(s.HeadStatus.Render("HEAD status: %s (State: %s)"), vmInfo.DisplayName, vmInfo.State)
	}
	fmt.Println()
	return nil
}

func emptyEnumerator(_ list.Items, _ int) string {
	return ""
}

func init() {
	rootCmd.AddCommand(statusCmd)
	statusCmd.Flags().StringP("cluster", "c", "", "Cluster ID or alias to show detailed status for")
}
