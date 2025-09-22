package cmd

import (
	"context"

	"github.com/hdresearch/vers-cli/internal/handlers"
	pres "github.com/hdresearch/vers-cli/internal/presenters"
	"github.com/spf13/cobra"
)

// pauseCmd represents the pause command
var pauseCmd = &cobra.Command{
	Use:   "pause [vm-id|alias]",
	Short: "Pause a running VM",
	Long:  `Pause a running Vers VM. If no VM ID or alias is provided, uses the current HEAD.`,
	Args:  cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		apiCtx, cancel := context.WithTimeout(context.Background(), application.Timeouts.APIMedium)
		defer cancel()
		var target string
		if len(args) > 0 {
			target = args[0]
		}
		view, err := handlers.HandlePause(apiCtx, application, handlers.PauseReq{Target: target})
		if err != nil {
			return err
		}
		pres.RenderPause(application, view)
		return nil
	},
}

func init() {
	rootCmd.AddCommand(pauseCmd)
}
