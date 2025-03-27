package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

var detach bool

// upCmd represents the up command
var upCmd = &cobra.Command{
	Use:   "up [cluster]",
	Short: "Start a development environment",
	Long:  `Start a Vers development environment according to the configuration in vers.toml.`,
	Args:  cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		// If no cluster name is provided, use "default"
		clusterName := "default"
		if len(args) > 0 {
			clusterName = args[0]
		}

		// Print startup message
		fmt.Printf("Starting cluster: %s\n", clusterName)
		if detach {
			fmt.Println("Running in detached mode")
		}

		// Call the SDK to start the cluster
		if err := client.StartCluster(clusterName); err != nil {
			return fmt.Errorf("failed to start cluster: %w", err)
		}

		fmt.Printf("Cluster %s started successfully\n", clusterName)
		return nil
	},
}

func init() {
	rootCmd.AddCommand(upCmd)

	// Define flags for the up command
	upCmd.Flags().BoolVarP(&detach, "detach", "d", false, "Run in detached mode")
} 