package cmd

import (
	"context"
	"fmt"
	"time"

	vers "github.com/hdresearch/vers-sdk-go"
	"github.com/spf13/cobra"
)

var clustersCmd = &cobra.Command{
	Use:   "clusters",
	Short: "List available Vers clusters",
	Long:  `Retrieves and displays a list of all available Vers clusters associated with your account.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		fmt.Println("Fetching list of clusters...")

		baseCtx := context.Background()
		client := vers.NewClient() // Assuming NewClient initializes appropriately

		apiCtx, cancel := context.WithTimeout(baseCtx, 30*time.Second)
		defer cancel()

		clusters, err := client.API.Cluster.List(apiCtx) // Placeholder for actual List parameters if any
		if err != nil {
			return fmt.Errorf("failed to list clusters: %w", err)
		}

		if clusters == nil || len(*clusters) == 0 { // Check for nil pointer and then dereference for length
			fmt.Println("No clusters found.")
			return nil
		}

		fmt.Println("Available clusters:")
		for _, cluster := range *clusters { // Dereference for ranging
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
	rootCmd.AddCommand(clustersCmd)
}
