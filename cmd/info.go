package cmd

import (
	"context"

	"github.com/hdresearch/vers-cli/internal/handlers"
	pres "github.com/hdresearch/vers-cli/internal/presenters"
	"github.com/spf13/cobra"
)

var (
	infoQuiet  bool
	infoFormat string
)

var infoCmd = &cobra.Command{
	Use:   "info [vm-id|alias]",
	Short: "Show detailed metadata for a VM",
	Long: `Display detailed metadata for a VM including IP address, lineage (parent commit,
grandparent VM), and timestamps. If no VM is specified, uses the current HEAD.

Use -q/--quiet to output just the VM ID.
Use --format json for machine-readable output.`,
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

		format := pres.ParseFormat(infoQuiet, infoFormat)
		switch format {
		case pres.FormatQuiet:
			pres.PrintQuiet([]string{res.Metadata.VmID})
		case pres.FormatJSON:
			pres.PrintJSON(res.Metadata)
		default:
			pres.RenderInfo(application, res)
		}
		return nil
	},
}

func init() {
	rootCmd.AddCommand(infoCmd)
	infoCmd.Flags().BoolVarP(&infoQuiet, "quiet", "q", false, "Only display VM ID")
	infoCmd.Flags().StringVar(&infoFormat, "format", "", "Output format (json)")
}
