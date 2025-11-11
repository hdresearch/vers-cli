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
	Short: "Get status of VMs",
	Long:  `Displays the status of all VMs by default. Provide a VM ID or alias as argument for VM-specific status.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		var target string
		if len(args) > 0 {
			target = args[0]
		}

		apiCtx, cancel := context.WithTimeout(context.Background(), application.Timeouts.APIMedium)
		defer cancel()

		res, err := handlers.HandleStatus(apiCtx, application, handlers.StatusReq{Target: target})
		if err != nil {
			return err
		}
		pres.RenderStatus(application, res)
		return nil
	},
}

func init() {
	rootCmd.AddCommand(statusCmd)
}
