package cmd

import (
	"context"
	"fmt"
	"io"
	"os"
	"time"

	"github.com/hdresearch/vers-cli/internal/handlers"
	pres "github.com/hdresearch/vers-cli/internal/presenters"
	"github.com/hdresearch/vers-cli/internal/utils"
	"github.com/spf13/cobra"
)

var executeTimeout int
var executeSSH bool
var executeWorkDir string
var executeStdin bool

// executeCmd represents the execute command
var executeCmd = &cobra.Command{
	Use:     "exec [vm-id|alias] <command> [args...]",
	Aliases: []string{"execute"},
	Short:   "Run a command on a specific VM",
	Long: `Execute a command on the specified VM via the orchestrator API.

The command runs through the in-VM agent, which means it automatically
inherits environment variables configured for your account.

If no VM is specified, the current HEAD VM is used.

Use -i to pass stdin from the local terminal to the remote command.
This is useful for piping data into commands, e.g.:

  echo '{"jsonrpc":"2.0","method":"ping","id":1}' | vers exec -i <vm> my-server

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
		// If there are multiple args, check if the first arg looks like a VM
		// identifier (UUID or known alias). If so, treat it as the target;
		// otherwise treat all args as the command and use HEAD.
		var target string
		var command []string

		if len(args) == 1 {
			target = ""
			command = args
		} else if utils.LooksLikeVMTarget(args[0]) {
			target = args[0]
			command = args[1:]
		} else {
			target = ""
			command = args
		}

		var timeoutSec uint64
		if executeTimeout > 0 {
			timeoutSec = uint64(executeTimeout)
		}

		// Read stdin if -i flag is set
		var stdinData string
		if executeStdin {
			data, err := io.ReadAll(os.Stdin)
			if err != nil {
				return fmt.Errorf("failed to read stdin: %w", err)
			}
			stdinData = string(data)
		}

		view, err := handlers.HandleExecute(apiCtx, application, handlers.ExecuteReq{
			Target:     target,
			Command:    command,
			WorkingDir: executeWorkDir,
			TimeoutSec: timeoutSec,
			UseSSH:     executeSSH,
			Stdin:      stdinData,
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
	executeCmd.Flags().BoolVar(&executeSSH, "ssh", false, "Use direct SSH instead of the VERS API")
	executeCmd.Flags().StringVarP(&executeWorkDir, "workdir", "w", "", "Working directory for the command")
	executeCmd.Flags().BoolVarP(&executeStdin, "interactive", "i", false, "Pass stdin to the remote command")
}
