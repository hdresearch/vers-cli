package cmd

import (
	"context"
	"fmt"
	"time"

	vers "github.com/hdresearch/vers-sdk-go"
	"github.com/spf13/cobra"
)

// statusCmd represents the status command
var statusCmd = &cobra.Command{
	Use:   "status",
	Short: "Get status of clusters or VMs",
	Long:  `Displays the status of all clusters or details of a specific cluster if specified with -cluster or -c flag.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		clusterID, _ := cmd.Flags().GetString("cluster")
		
		baseCtx := context.Background()
		client := vers.NewClient()
		apiCtx, cancel := context.WithTimeout(baseCtx, 30*time.Second)
		defer cancel()

		// If cluster flag is provided, show status for that specific cluster
		if clusterID != "" {
			fmt.Printf("Getting status for cluster: %s\n", clusterID)

			// Call the Get cluster endpoint with the cluster ID
			fmt.Println("Fetching cluster information...")
			cluster, err := client.API.Cluster.Get(apiCtx, clusterID)
			if err != nil {
				return fmt.Errorf("failed to get status for cluster '%s': %w", clusterID, err)
			}

			// Display the cluster information
			fmt.Printf("Cluster ID: %s\n", cluster.ID)
			fmt.Println("VMs in this cluster:")
			fmt.Println("-----------------------")
			
			if len(cluster.Children) == 0 {
				fmt.Println("No VMs found in this cluster.")
			} else {
				// Display each VM's information
				for _, vm := range cluster.Children {
					fmt.Printf("VM ID: %s\n", vm.ID)
					fmt.Printf("  State: %s\n", vm.State)
					fmt.Printf("  IP Address: %s\n", vm.IPAddress)
					fmt.Println()
				}
			}
			
			return nil
		}

		// If no cluster ID provided, list all clusters
		fmt.Println("Fetching list of clusters...")

		clusters, err := client.API.Cluster.List(apiCtx)
		if err != nil {
			return fmt.Errorf("failed to list clusters: %w", err)
		}

		if clusters == nil || len(*clusters) == 0 {
			fmt.Println("No clusters found.")
			return nil
		}

		fmt.Println("Available clusters:")
		for _, cluster := range *clusters {
			fmt.Printf("  - ID: %s, Root VM ID: %s, Children: %d\n",
				cluster.ID, 
				cluster.RootVmID,         
				cluster.VmCount,       
			)
		}

		return nil
	},
}

func init() {
	rootCmd.AddCommand(statusCmd)
	statusCmd.Flags().StringP("cluster", "c", "", "Cluster ID to show detailed status for")
}