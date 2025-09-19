package cmd

import (
	"context"

	"github.com/hdresearch/vers-cli/internal/handlers"
	pres "github.com/hdresearch/vers-cli/internal/presenters"
	"github.com/spf13/cobra"
)

// executeCmd represents the execute command
var executeCmd = &cobra.Command{
	Use:   "execute [vm-id|alias] [args...]",
	Short: "Run a command on a specific VM",
	Long:  `Execute a command within the Vers environment on the specified VM. If no VM ID or alias is provided, uses the current HEAD.`,
	Args:  cobra.MinimumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		apiCtx, cancel := context.WithTimeout(context.Background(), application.Timeouts.APIMedium)
		defer cancel()
		// If more than one arg: treat first as potential target; handlers mirror original behavior
		var target string
		var command []string
		if len(args) > 1 {
			target, command = args[0], args[1:]
		} else {
			command = args
		}
		view, err := handlers.HandleExecute(apiCtx, application, handlers.ExecuteReq{Target: target, Command: command})
		if err != nil {
			return err
		}
		pres.RenderExecute(application, view)
		return nil
	},
}

func init() {
	rootCmd.AddCommand(executeCmd)
	executeCmd.Flags().String("host", "", "Specify the host IP to connect to (overrides default)")
}
