package cmd

import (
	"context"

	"github.com/hdresearch/vers-cli/internal/handlers"
	pres "github.com/hdresearch/vers-cli/internal/presenters"
	"github.com/spf13/cobra"
)

// statusCmd represents the status command
var statusCmd = &cobra.Command{
	Use:   "status [vm-id|alias]",
	Short: "Get status of clusters or VMs",
	Long:  `Displays the status of all clusters by default. Use -c flag for specific cluster details, or provide a VM ID or alias as argument for VM-specific status.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		clusterID, _ := cmd.Flags().GetString("cluster")
		var target string
		if len(args) > 0 {
			target = args[0]
		}

		apiCtx, cancel := context.WithTimeout(context.Background(), application.Timeouts.APIMedium)
		defer cancel()

		res, err := handlers.HandleStatus(apiCtx, application, handlers.StatusReq{Cluster: clusterID, Target: target})
		if err != nil {
			return err
		}
		pres.RenderStatus(application, res)
		return nil
	},
}

// Handle cluster status with single API call
// No additional helpers; rendering handled by presenters.

func init() {
	rootCmd.AddCommand(statusCmd)
	statusCmd.Flags().StringP("cluster", "c", "", "Cluster ID or alias to show detailed status for")
}
