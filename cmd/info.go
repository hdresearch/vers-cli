package cmd

import (
	"context"

	"github.com/hdresearch/vers-cli/internal/handlers"
	pres "github.com/hdresearch/vers-cli/internal/presenters"
	"github.com/spf13/cobra"
)

var infoCmd = &cobra.Command{
	Use:   "info [vm-id|alias]",
	Short: "Show detailed metadata for a VM",
	Long: `Display detailed metadata for a VM including IP address, lineage (parent commit,
grandparent VM), and timestamps. If no VM is specified, uses the current HEAD.`,
	Args: cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		target := ""
		if len(args) > 0 {
			target = args[0]
		}

		apiCtx, cancel := context.WithTimeout(context.Background(), application.Timeouts.APIMedium)
		defer cancel()

		res, err := handlers.HandleInfo(apiCtx, application, handlers.InfoReq{Target: target})
		if err != nil {
			return err
		}
		pres.RenderInfo(application, res)
		return nil
	},
}

func init() {
	rootCmd.AddCommand(infoCmd)
}
