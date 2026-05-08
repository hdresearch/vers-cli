package cmd

import (
	"context"

	"github.com/hdresearch/vers-cli/internal/handlers"
	pres "github.com/hdresearch/vers-cli/internal/presenters"
	"github.com/spf13/cobra"
)

var (
	resumeJSON   bool
	resumeFormat string
	resumeWait   bool
)

var resumeCmd = &cobra.Command{
	Use:   "resume [vm-id|alias]",
	Short: "Resume a paused VM",
	Long: `Resume a paused Vers VM. If no VM ID or alias is provided, uses the current HEAD.

Use --json for machine-readable output.
Use --wait to block until the VM is running.`,
	Args: cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		apiCtx, cancel := context.WithTimeout(context.Background(), application.Timeouts.APILong)
		defer cancel()
		var target string
		if len(args) > 0 {
			target = args[0]
		}
		view, err := handlers.HandleResume(apiCtx, application, handlers.ResumeReq{
			Target: target,
			Wait:   resumeWait,
		})
		if err != nil {
			return err
		}

		format, err := pres.ParseFormat(false, resumeJSON, resumeFormat)
		if err != nil {
			return err
		}
		switch format {
		case pres.FormatJSON:
			pres.PrintJSON(map[string]string{"vm_id": view.VMName, "state": view.NewState})
		default:
			pres.RenderResume(application, view)
		}
		return nil
	},
}

func init() {
	rootCmd.AddCommand(resumeCmd)
	resumeCmd.Flags().BoolVar(&resumeJSON, "json", false, "Output as JSON")
	resumeCmd.Flags().StringVar(&resumeFormat, "format", "", "Output format (json) [deprecated: use --json]")
	_ = resumeCmd.Flags().MarkDeprecated("format", "use --json instead")
	resumeCmd.Flags().BoolVar(&resumeWait, "wait", false, "Wait until VM is running")
}
