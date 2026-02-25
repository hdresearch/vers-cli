package cmd

import (
	"context"
	"time"

	"github.com/hdresearch/vers-cli/internal/handlers"
	pres "github.com/hdresearch/vers-cli/internal/presenters"
	"github.com/spf13/cobra"
)

// executeCmd represents the execute command
var executeCmd = &cobra.Command{
	Use:   "execute [vm-id|alias] <command> [args...]",
	Short: "Run a command on a specific VM",
	Long: `Execute a command within the Vers environment on the specified VM.
If no VM is specified, the current HEAD VM is used.`,
	Args: cobra.MinimumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		// Get timeout from flag, default to APIMedium (30s)
		timeoutSec, _ := cmd.Flags().GetInt("timeout")
		var timeout time.Duration
		if timeoutSec > 0 {
			timeout = time.Duration(timeoutSec) * time.Second
		} else {
			timeout = application.Timeouts.APIMedium
		}

		apiCtx, cancel := context.WithTimeout(context.Background(), timeout)
		defer cancel()

		// Determine if the first arg is a VM target or part of the command.
		// If there's only one arg, it's the command (use HEAD VM).
		// If there are multiple args, try to resolve the first arg as a VM;
		// if it resolves, treat it as the target, otherwise treat all args as the command.
		var target string
		var command []string

		if len(args) == 1 {
			// Only a command, use HEAD VM
			target = ""
			command = args
		} else {
			// First arg is the target VM, remaining args are the command
			target = args[0]
			command = args[1:]
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
	executeCmd.Flags().IntP("timeout", "t", 0, "Timeout in seconds (default: 30s, use 0 for no limit)")
}
