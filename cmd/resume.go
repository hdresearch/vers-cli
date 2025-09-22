package cmd

import (
	"context"

	"github.com/hdresearch/vers-cli/internal/handlers"
	pres "github.com/hdresearch/vers-cli/internal/presenters"
	"github.com/spf13/cobra"
)

// resumeCmd represents the resume command
var resumeCmd = &cobra.Command{
	Use:   "resume [vm-id|alias]",
	Short: "Resume a paused VM",
	Long:  `Resume a paused Vers VM. If no VM ID or alias is provided, uses the current HEAD.`,
	Args:  cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		apiCtx, cancel := context.WithTimeout(context.Background(), application.Timeouts.APIMedium)
		defer cancel()
		var target string
		if len(args) > 0 {
			target = args[0]
		}
		view, err := handlers.HandleResume(apiCtx, application, handlers.ResumeReq{Target: target})
		if err != nil {
			return err
		}
		pres.RenderResume(application, view)
		return nil
	},
}

func init() {
	rootCmd.AddCommand(resumeCmd)
}
