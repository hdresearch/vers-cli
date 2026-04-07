package cmd

import (
	"context"
	"os"
	"time"

	"github.com/hdresearch/vers-cli/internal/handlers"
	pres "github.com/hdresearch/vers-cli/internal/presenters"
	"github.com/spf13/cobra"
)

var executeTimeout int
var executeSSH bool
var executeWorkDir string

// executeCmd represents the execute command
var executeCmd = &cobra.Command{
	Use:     "execute [vm-id|alias] <command> [args...]",
	Aliases: []string{"exec"},
	Short:   "Run a command on a specific VM",
	Long: `Execute a command on the specified VM via the orchestrator API.

The command runs through the in-VM agent, which means it automatically
inherits environment variables and secrets configured for your account.

If no VM is specified, the current HEAD VM is used.

Use --ssh to bypass the API and connect directly via SSH (legacy behavior).`,
	Args: cobra.MinimumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		// Use custom timeout if specified, otherwise use default APIMedium
		timeout := application.Timeouts.APIMedium
		if executeTimeout > 0 {
			timeout = time.Duration(executeTimeout) * time.Second
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

		var timeoutSec uint64
		if executeTimeout > 0 {
			timeoutSec = uint64(executeTimeout)
		}

		view, err := handlers.HandleExecute(apiCtx, application, handlers.ExecuteReq{
			Target:     target,
			Command:    command,
			WorkingDir: executeWorkDir,
			TimeoutSec: timeoutSec,
			UseSSH:     executeSSH,
		})
		if err != nil {
			return err
		}
		pres.RenderExecute(application, view)

		// Exit with the command's exit code
		if view.ExitCode != 0 {
			os.Exit(view.ExitCode)
		}
		return nil
	},
}

func init() {
	rootCmd.AddCommand(executeCmd)
	executeCmd.Flags().SetInterspersed(false) // stop flag parsing after first positional arg
	executeCmd.Flags().IntVarP(&executeTimeout, "timeout", "t", 0, "Timeout in seconds (default: 30s, use 0 for no limit)")
	executeCmd.Flags().BoolVar(&executeSSH, "ssh", false, "Use direct SSH instead of the orchestrator API")
	executeCmd.Flags().StringVarP(&executeWorkDir, "workdir", "w", "", "Working directory for the command")
}
