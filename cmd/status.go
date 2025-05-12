package cmd

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/charmbracelet/lipgloss/list"
	"github.com/hdresearch/vers-cli/styles"
	"github.com/spf13/cobra"
)

// statusCmd represents the status command
var statusCmd = &cobra.Command{
	Use:   "status",
	Short: "Get status of clusters or VMs",
	Long:  `Displays the status of all clusters or details of a specific cluster if specified with -cluster or -c flag.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		// Display current HEAD information

		clusterID, _ := cmd.Flags().GetString("cluster")

		baseCtx := context.Background()
		apiCtx, cancel := context.WithTimeout(baseCtx, 30*time.Second)
		defer cancel()

		s := styles.NewStatusStyles()

		// If cluster flag is provided, show status for that specific cluster
		if clusterID != "" {
			fmt.Printf(s.HeadStatus.Render("Getting status for cluster: "+clusterID) + "\n")

			// Call the Get cluster endpoint with the cluster ID
			fmt.Println(s.NoData.Render("Fetching cluster information..."))
			response, err := client.API.Cluster.Get(apiCtx, clusterID)
			if err != nil {
				return fmt.Errorf(styles.ErrorTextStyle.Render("failed to get status for cluster '%s': %w"), clusterID, err)
			}
			cluster := response.Data

			displayHeadStatus()

			fmt.Println(s.VMListHeader.Render("Cluster details:"))
			clusterList := list.New().Enumerator(emptyEnumerator).ItemStyle(s.ClusterListItem)

			// Format cluster info similar to the default view
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

		// If no cluster ID provided, list all clusters
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

		displayHeadStatus()

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

		tip := "\nTip: To view the list of VMs in a specific cluster, use: vers status -c <cluster-id>"
		fmt.Println(s.Tip.Render(tip))

		return nil
	},
}

// Helper function to display current HEAD status
func displayHeadStatus() error {
	versDir := ".vers"
	headFile := filepath.Join(versDir, "HEAD")

	s := styles.NewStatusStyles()

	// Check if .vers directory and HEAD file exist
	if _, err := os.Stat(headFile); os.IsNotExist(err) {
		return fmt.Errorf(s.HeadStatus.Render("HEAD status: Not a vers repository (or run 'vers init' first)"))
	}

	// Read HEAD file
	headData, err := os.ReadFile(headFile)
	if err != nil {
		return fmt.Errorf(styles.ErrorTextStyle.Render("HEAD status: Error reading HEAD file (%w)\n"), err)
	}

	// Parse the HEAD content
	headContent := string(bytes.TrimSpace(headData))

	// Check if HEAD is a symbolic ref or direct ref
	var headStatus string
	if strings.HasPrefix(headContent, "ref: ") {
		// It's a symbolic ref, extract the branch name
		refPath := strings.TrimPrefix(headContent, "ref: ")
		branchName := strings.TrimPrefix(refPath, "refs/heads/")

		// Read the actual reference file to get VM ID
		refFile := filepath.Join(versDir, refPath)
		vmID := "unknown"

		if refData, err := os.ReadFile(refFile); err == nil {
			vmID = string(bytes.TrimSpace(refData))
		}

		headStatus = fmt.Sprintf(s.HeadStatus.Render("HEAD status: On branch '%s' (VM: %s)"), branchName, vmID)
	} else {
		// HEAD directly contains a VM ID (detached HEAD state)
		headStatus = fmt.Sprintf("HEAD status: Detached HEAD at VM '%s'", headContent)
	}

	fmt.Println(s.HeadStatus.Render(headStatus))
	return nil
}

func emptyEnumerator(_ list.Items, _ int) string {
	return ""
}

func init() {
	rootCmd.AddCommand(statusCmd)
	statusCmd.Flags().StringP("cluster", "c", "", "Cluster ID to show detailed status for")
}
