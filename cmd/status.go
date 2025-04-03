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
	Use:   "status <cluster-id>",
	Short: "Get status of a cluster",
	Long:  `Displays the status of a cluster by showing all VMs in the cluster.`,
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		clusterID := args[0]
		
		fmt.Printf("Getting status for cluster: %s\n", clusterID)

		baseCtx := context.Background()
		client = vers.NewClient()
		apiCtx, cancel := context.WithTimeout(baseCtx, 30*time.Second)
		defer cancel() 

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
	},
}

func init() {
	rootCmd.AddCommand(statusCmd)
}