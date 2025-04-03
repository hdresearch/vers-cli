package cmd

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	vers "github.com/hdresearch/vers-sdk-go"
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
		// Generate a cluster name with UUID suffix
		clusterName := fmt.Sprintf("new-cluster-%s", uuid.New() )
		if len(args) > 0 {
			clusterName = args[0]
		}
		if(detach) { 
			fmt.Printf("Running in detached mode\n")
		}

		fmt.Printf("Preparing cluster parameters for cluster: %s\n", clusterName)

		baseCtx := context.Background()
		
		// Call the SDK to start the cluster
		client = vers.NewClient()

		apiCtx, cancel := context.WithTimeout(baseCtx, 30*time.Second)
		defer cancel() 

		clusterParams := vers.APIClusterNewParams {
			Body: map[string]interface{}{
				"name": clusterName,
			},
		}

		fmt.Println("Sending request to start cluster...")
		clusterInfo, err := client.API.Cluster.New(apiCtx, clusterParams)
		if err != nil {
			return fmt.Errorf("failed to start cluster '%s': %w", clusterName, err)
		}
				// Use information from the response (adjust field names as needed)
				fmt.Printf("Cluster '%s' (ID: %s) started successfully using image '%s'.\n",
				clusterInfo.ID,
				clusterInfo.ID,
				clusterInfo.RootVmID,
			)
			return nil
	},
}

func init() {
	rootCmd.AddCommand(upCmd)

	// Define flags for the up command
	upCmd.Flags().BoolVarP(&detach, "detach", "d", false, "Run in detached mode")
} 