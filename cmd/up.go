package cmd

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"time"

	vers "github.com/hdresearch/vers-sdk-go"
	"github.com/spf13/cobra"
)

// upCmd represents the up command
var upCmd = &cobra.Command{
	Use:   "up [cluster]",
	Short: "Start a development environment",
	Long:  `Start a Vers development environment according to the configuration in vers.toml.`,
	Args:  cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		baseCtx := context.Background()

		apiCtx, cancel := context.WithTimeout(baseCtx, 30*time.Second)
		defer cancel()

		clusterParams := vers.APIClusterNewParams{}

		fmt.Println("Sending request to start cluster...")
		clusterInfo, err := client.API.Cluster.New(apiCtx, clusterParams)
		if err != nil {
			return fmt.Errorf("failed to start cluster: %w", err)
		}
		// Use information from the response (adjust field names as needed)
		fmt.Printf("Cluster (ID: %s) started successfully with root vm '%s'.\n",
			clusterInfo.ID,
			clusterInfo.RootVmID,
		)

		// Store VM ID in version control system
		vmID := clusterInfo.RootVmID
		if vmID != "" {
			// Check if .vers directory exists
			versDir := ".vers"
			if _, err := os.Stat(versDir); os.IsNotExist(err) {
				fmt.Println("Warning: .vers directory not found. Run 'vers init' first.")
			} else {
				// Update refs/heads/main with VM ID
				mainRefPath := filepath.Join(versDir, "refs", "heads", "main")
				if err := os.WriteFile(mainRefPath, []byte(vmID+"\n"), 0644); err != nil {
					fmt.Printf("Warning: Failed to update refs: %v\n", err)
				} else {
					fmt.Printf("Updated VM reference: %s -> %s\n", "refs/heads/main", vmID)
				}

				// HEAD already points to refs/heads/main from init, so we don't need to update it
				fmt.Println("HEAD is now pointing to the new VM")
			}
		}

		return nil
	},
}

func init() {
	rootCmd.AddCommand(upCmd)
}
