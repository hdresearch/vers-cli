package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

var stopAll bool

// stopCmd represents the stop command
var stopCmd = &cobra.Command{
	Use:   "stop [cluster]",
	Short: "Stop a running development environment",
	Long:  `Stop a running Vers development environment gracefully.`,
	Args:  cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		// If no cluster name is provided and not stopping all, use "default"
		clusterName := "default"
		if len(args) > 0 {
			clusterName = args[0]
		}

		// Print stop message
		if stopAll {
			fmt.Println("Stopping all clusters")
		} else {
			fmt.Printf("Stopping cluster: %s\n", clusterName)
		}

		// Call the SDK to stop the cluster
		if err := client.StopCluster(clusterName, stopAll); err != nil {
			return fmt.Errorf("failed to stop cluster: %w", err)
		}

		if stopAll {
			fmt.Println("All clusters stopped successfully")
		} else {
			fmt.Printf("Cluster %s stopped successfully\n", clusterName)
		}
		return nil
	},
}

func init() {
	rootCmd.AddCommand(stopCmd)

	// Define flags for the stop command
	stopCmd.Flags().BoolVarP(&stopAll, "all", "a", false, "Stop all running clusters")
} 