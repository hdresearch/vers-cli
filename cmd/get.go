package cmd

import (
	"context"

	"github.com/hdresearch/vers-cli/internal/handlers"
	pres "github.com/hdresearch/vers-cli/internal/presenters"
	"github.com/spf13/cobra"
)

var (
	getQuiet  bool
	getJSON bool
	getFormat string
)

var getCmd = &cobra.Command{
	Use:     "get [vm-id|alias]",
	Aliases: []string{"info"},
	Short:   "Show detailed metadata for a VM",
	Long: `Display detailed metadata for a VM including IP address, lineage (parent commit,
grandparent VM), and timestamps. If no VM is specified, uses the current HEAD.

Use -q/--quiet to output just the VM ID.
Use --json for machine-readable output.

The "info" alias is preserved for backwards compatibility.`,
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

		format, err := pres.ParseFormat(getQuiet, getJSON, getFormat)
		if err != nil {
			return err
		}
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
	rootCmd.AddCommand(getCmd)
	getCmd.Flags().BoolVarP(&getQuiet, "quiet", "q", false, "Only display VM ID")
	getCmd.Flags().BoolVar(&getJSON, "json", false, "Output as JSON")
	getCmd.Flags().StringVar(&getFormat, "format", "", "Output format (json) [deprecated: use --json]")
	_ = getCmd.Flags().MarkDeprecated("format", "use --json instead")
}
