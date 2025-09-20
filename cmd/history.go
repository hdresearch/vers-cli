package cmd

import (
	"context"

	"github.com/hdresearch/vers-cli/internal/handlers"
	pres "github.com/hdresearch/vers-cli/internal/presenters"
	"github.com/spf13/cobra"
)

// logCmd represents the log command
var logCmd = &cobra.Command{
	Use:   "history [vm-id|alias]",
	Short: "Display commit history",
	Long:  `Shows the commit history for the current VM or a specified VM ID or alias.`,
	Args:  cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		apiCtx, cancel := context.WithTimeout(context.Background(), application.Timeouts.APIMedium)
		defer cancel()
		var target string
		if len(args) > 0 {
			target = args[0]
		}
		view, err := handlers.HandleHistory(apiCtx, application, handlers.HistoryReq{Target: target})
		if err != nil {
			return err
		}
		pres.RenderHistory(application, view)
		return nil
	},
}

func init() {
	rootCmd.AddCommand(logCmd)
}
