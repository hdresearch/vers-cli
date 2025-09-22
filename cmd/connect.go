package cmd

import (
	"context"

	"github.com/hdresearch/vers-cli/internal/handlers"
	pres "github.com/hdresearch/vers-cli/internal/presenters"
	"github.com/spf13/cobra"
)

// connectCmd represents the connect command
var connectCmd = &cobra.Command{
	Use:   "connect [vm-id|alias]",
	Short: "Connect to a VM via SSH",
	Long:  `Connect to a running Vers VM via SSH. If no VM ID or alias is provided, uses the current HEAD.`,
	Args:  cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		apiCtx, cancel := context.WithTimeout(context.Background(), application.Timeouts.APIMedium)
		defer cancel()
		target := ""
		if len(args) > 0 {
			target = args[0]
		}
		view, err := handlers.HandleConnect(apiCtx, application, handlers.ConnectReq{Target: target})
		if err != nil {
			return err
		}
		pres.RenderConnect(application, view)
		return nil
	},
}

func init() {
	rootCmd.AddCommand(connectCmd)
	connectCmd.Flags().String("host", "", "Specify the host IP to connect to (overrides default)")
}
