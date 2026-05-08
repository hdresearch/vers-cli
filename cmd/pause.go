package cmd

import (
	"context"

	"github.com/hdresearch/vers-cli/internal/handlers"
	pres "github.com/hdresearch/vers-cli/internal/presenters"
	"github.com/spf13/cobra"
)

var pauseJSON bool
var pauseFormat string

var pauseCmd = &cobra.Command{
	Use:   "pause [vm-id|alias]",
	Short: "Pause a running VM",
	Long: `Pause a running Vers VM. If no VM ID or alias is provided, uses the current HEAD.

Use --json for machine-readable output.`,
	Args: cobra.MaximumNArgs(1),
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

		format, err := pres.ParseFormat(false, pauseJSON, pauseFormat)
		if err != nil {
			return err
		}
		switch format {
		case pres.FormatJSON:
			pres.PrintJSON(map[string]string{"vm_id": view.VMName, "state": view.NewState})
		default:
			pres.RenderPause(application, view)
		}
		return nil
	},
}

func init() {
	rootCmd.AddCommand(pauseCmd)
	pauseCmd.Flags().BoolVar(&pauseJSON, "json", false, "Output as JSON")
	pauseCmd.Flags().StringVar(&pauseFormat, "format", "", "Output format (json) [deprecated: use --json]")
	_ = pauseCmd.Flags().MarkDeprecated("format", "use --json instead")
}
