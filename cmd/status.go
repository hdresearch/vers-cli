package cmd

import (
    "context"
    "fmt"
    "strings"
    "time"

    "github.com/charmbracelet/lipgloss/list"
    "github.com/hdresearch/vers-cli/internal/presenters"
    svc "github.com/hdresearch/vers-cli/internal/services/status"
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
    cluster, err := svc.GetCluster(ctx, client, clusterID)
    if err != nil {
        return fmt.Errorf(styles.ErrorTextStyle.Render("failed to find cluster '%s': %w"), clusterID, err)
    }
    presenters.RenderClusterStatus(s, cluster)
    return nil
}

// Handle VM status with single API call
func handleVMStatus(ctx context.Context, vmIdentifier string, s *styles.StatusStyles) error {
    vm, err := svc.GetVM(ctx, client, vmIdentifier)
    if err != nil {
        return fmt.Errorf(styles.ErrorTextStyle.Render("failed to find VM '%s': %w"), vmIdentifier, err)
    }
    presenters.RenderVMStatus(s, vm)
    return nil
}

// Handle default status (list all clusters)
func handleDefaultStatus(ctx context.Context, s *styles.StatusStyles) error {
    // List all clusters
    fmt.Println(s.NoData.Render("Fetching list of clusters..."))
    clusters, err := svc.ListClusters(ctx, client)
    if err != nil {
        return fmt.Errorf(styles.ErrorTextStyle.Render("failed to list clusters: %w"), err)
    }

    if len(clusters) == 0 {
        fmt.Println(s.NoData.Render("No clusters found."))
        return nil
    }

    presenters.RenderClusterList(s, clusters)

    tip := "\nTip: To view VMs in a specific cluster, use: vers status -c <cluster-id>\n" +
        "To view a specific VM, use: vers status <vm-id>"
    fmt.Println(s.Tip.Render(tip))

    return nil
}

// Only show HEAD status when relevant (default case only)
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
	apiCtx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
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
