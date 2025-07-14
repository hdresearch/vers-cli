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

		// Only show HEAD status for default case (no specific target requested)
		showHeadStatus := clusterID == "" && len(args) == 0
		if showHeadStatus {
			displayHeadStatus()
		}

		// If cluster flag is provided, show status for that specific cluster
		if clusterID != "" {
			return handleClusterStatus(apiCtx, clusterID, &s)
		}

		// If VM ID is provided as argument, show status for that specific VM
		if len(args) > 0 {
			return handleVMStatus(apiCtx, args[0], &s)
		}

		// If no cluster ID or VM ID provided, list all clusters
		return handleDefaultStatus(apiCtx, &s)
	},
}

// Handle cluster status with single API call
func handleClusterStatus(ctx context.Context, clusterID string, s *styles.StatusStyles) error {
	// Single API call - Cluster.Get() can resolve by ID or alias directly
	response, err := client.API.Cluster.Get(ctx, clusterID)
	if err != nil {
		return fmt.Errorf(styles.ErrorTextStyle.Render("failed to find cluster '%s': %w"), clusterID, err)
	}

	cluster := response.Data

	// Find the root VM in the VMs list to get its alias
	var rootVMAlias string
	for _, vm := range cluster.Vms {
		if vm.ID == cluster.RootVmID {
			rootVMAlias = vm.Alias
			break
		}
	}

	// Create display name (prefer alias over ID)
	clusterDisplayName := cluster.Alias
	if clusterDisplayName == "" {
		clusterDisplayName = cluster.ID
	}

	// Create root VM display name (prefer alias over ID)
	rootVMDisplayName := rootVMAlias
	if rootVMDisplayName == "" {
		rootVMDisplayName = cluster.RootVmID
	}

	fmt.Printf(s.HeadStatus.Render("Getting status for cluster: "+clusterDisplayName) + "\n")

	fmt.Println(s.VMListHeader.Render("Cluster details:"))
	clusterList := list.New().Enumerator(emptyEnumerator).ItemStyle(s.ClusterListItem)

	// Format cluster info
	clusterInfo_display := fmt.Sprintf(
		"%s\n%s\n%s",
		s.ClusterName.Render("Cluster: "+clusterDisplayName),
		s.ClusterData.Render("Root VM: "+s.VMID.Render(rootVMDisplayName)),
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

// Handle VM status with single API call
func handleVMStatus(ctx context.Context, vmIdentifier string, s *styles.StatusStyles) error {
	// Single API call - Vm.Get() can resolve by ID or alias directly
	response, err := client.API.Vm.Get(ctx, vmIdentifier)
	if err != nil {
		return fmt.Errorf(styles.ErrorTextStyle.Render("failed to find VM '%s': %w"), vmIdentifier, err)
	}

	vm := response.Data

	// Create VMInfo from response
	vmInfo := utils.CreateVMInfoFromGetResponse(vm)

	fmt.Printf(s.HeadStatus.Render("Getting status for VM: "+vmInfo.DisplayName) + "\n")

	fmt.Println(s.VMListHeader.Render("VM details:"))
	vmList := list.New().Enumerator(emptyEnumerator).ItemStyle(s.ClusterListItem)

	// Use data from single API call - no second Vm.Get() needed
	vmInfo_display := fmt.Sprintf(
		"%s\n%s\n%s",
		s.ClusterName.Render("VM: "+s.VMID.Render(vmInfo.DisplayName)),
		s.ClusterData.Render("State: "+vmInfo.State),
		s.ClusterData.Render("Cluster: "+vm.ClusterID),
	)

	vmList.Items(vmInfo_display)
	fmt.Println(vmList)

	tip := "\nTip: To view the cluster containing this VM, run: vers status -c " + vm.ClusterID
	fmt.Println(s.Tip.Render(tip))

	return nil
}

// Handle default status (list all clusters)
func handleDefaultStatus(ctx context.Context, s *styles.StatusStyles) error {
	// List all clusters
	fmt.Println(s.NoData.Render("Fetching list of clusters..."))
	response, err := client.API.Cluster.List(ctx)
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

		// Try to find root VM alias in the Vms array (if populated)
		rootVMDisplayName := cluster.RootVmID // Default to ID
		for _, vm := range cluster.Vms {
			if vm.ID == cluster.RootVmID && vm.Alias != "" {
				rootVMDisplayName = vm.Alias
				break
			}
		}

		// Combine the cluster name and its data into a single string
		clusterInfo := fmt.Sprintf(
			"%s\n%s\n%s",
			s.ClusterName.Render("Cluster: "+displayName),
			s.ClusterData.Render("Root VM: "+s.VMID.Render(rootVMDisplayName)),
			s.ClusterData.Render("# children: "+fmt.Sprintf("%d", cluster.VmCount)),
		)
		clusterList.Items(clusterInfo)
	}
	fmt.Println(clusterList)

	tip := "\nTip: To view VMs in a specific cluster, use: vers status -c <cluster-id>\n" +
		"To view a specific VM, use: vers status <vm-id>"
	fmt.Println(s.Tip.Render(tip))

	return nil
}

// Only show HEAD status when relevant (default case only)
func displayHeadStatus() error {
	s := styles.NewStatusStyles()

	// Get HEAD display name directly using the new function
	headDisplayName, err := utils.GetCurrentHeadDisplayName()
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

	// Try to get VM details to show state
	headVMID, err := utils.GetCurrentHeadVM()
	if err != nil {
		// This shouldn't happen if GetCurrentHeadDisplayName worked, but fallback
		fmt.Printf(s.HeadStatus.Render("HEAD status: %s"), headDisplayName)
		fmt.Println()
		return nil
	}

	apiCtx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	response, err := client.API.Vm.Get(apiCtx, headVMID)
	if err != nil {
		fmt.Printf(s.HeadStatus.Render("HEAD status: %s (unable to verify)"), headDisplayName)
	} else {
		fmt.Printf(s.HeadStatus.Render("HEAD status: %s (State: %s)"), headDisplayName, response.Data.State)
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
