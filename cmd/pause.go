package cmd

import (
	"context"

	"github.com/hdresearch/vers-cli/internal/handlers"
	pres "github.com/hdresearch/vers-cli/internal/presenters"
	"github.com/spf13/cobra"
)

var pauseFormat string

var pauseCmd = &cobra.Command{
	Use:   "pause [vm-id|alias]",
	Short: "Pause a running VM",
	Long: `Pause a running Vers VM. If no VM ID or alias is provided, uses the current HEAD.

Use --format json for machine-readable output.`,
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
			trackSemanticOutcome("vm_paused", err, map[string]any{
				"target_provided": target != "",
			})
			return err
		}

		format := pres.ParseFormat(false, pauseFormat)
		switch format {
		case pres.FormatJSON:
			pres.PrintJSON(map[string]string{"vm_id": view.VMName, "state": view.NewState})
		default:
			pres.RenderPause(application, view)
		}
		trackSemanticEvent("vm_paused", map[string]any{
			"vm_id":           view.VMName,
			"target_provided": target != "",
		})
		return nil
	},
}

func init() {
	rootCmd.AddCommand(pauseCmd)
	pauseCmd.Flags().StringVar(&pauseFormat, "format", "", "Output format (json)")
}
