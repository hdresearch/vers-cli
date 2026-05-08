package cmd

import (
	"context"

	"github.com/hdresearch/vers-cli/internal/handlers"
	pres "github.com/hdresearch/vers-cli/internal/presenters"
	"github.com/spf13/cobra"
)

var (
	statusQuiet  bool
	statusJSON   bool
	statusFormat string
	statusLimit  int
	statusOffset int
)

// statusCmd represents the status command
var statusCmd = &cobra.Command{
	Use:   "status [vm-id|alias]",
	Short: "Get status of VMs",
	Long: `Displays the status of all VMs by default. Provide a VM ID or alias as argument for VM-specific status.

Use -q/--quiet to output just VM IDs (one per line), useful for scripting:
  vers kill $(vers status -q)              # kill all VMs
  vers get $(vers status -q | head -1)    # info on first VM

Use --json for machine-readable output.

Pagination (when listing VMs):
  --limit N    Cap results at N (default 50). Use 0 for unbounded.
  --offset N   Skip the first N results (use with --limit to page).

When the result is truncated, a hint with --offset for the next page is
printed to stderr (text mode) or included in the JSON envelope.`,
	Aliases: []string{"ps"},
	RunE: func(cmd *cobra.Command, args []string) error {
		var target string
		if len(args) > 0 {
			target = args[0]
		}

		apiCtx, cancel := context.WithTimeout(context.Background(), application.Timeouts.APIMedium)
		defer cancel()

		res, err := handlers.HandleStatus(apiCtx, application, handlers.StatusReq{Target: target})
		if err != nil {
			return err
		}

		format, err := pres.ParseFormat(statusQuiet, statusJSON, statusFormat)
		if err != nil {
			return err
		}

		// Single-VM mode: pagination does not apply.
		if res.Mode == pres.StatusVM && res.VM != nil {
			switch format {
			case pres.FormatQuiet:
				pres.PrintQuiet([]string{res.VM.VmID})
			case pres.FormatJSON:
				pres.PrintJSON(res.VM)
			default:
				pres.RenderStatus(application, res)
			}
			return nil
		}

		// List mode: apply client-side pagination over res.VMs.
		// TODO: when the SDK exposes server-side limit/offset on the VM list
		// endpoint, plumb statusLimit/statusOffset through to the API.
		start, end, info := pres.ApplyPaging(len(res.VMs), statusLimit, statusOffset)
		paged := res.VMs[start:end]

		switch format {
		case pres.FormatQuiet:
			ids := make([]string, len(paged))
			for i, vm := range paged {
				ids[i] = vm.VmID
			}
			pres.PrintQuiet(ids)
			pres.PrintTruncationHint(cmd.ErrOrStderr(), info)
		case pres.FormatJSON:
			pres.PrintListJSON(paged, info)
		default:
			pagedView := res
			pagedView.VMs = paged
			pres.RenderStatus(application, pagedView)
			pres.PrintTruncationHint(cmd.ErrOrStderr(), info)
		}
		return nil
	},
}

func init() {
	rootCmd.AddCommand(statusCmd)
	statusCmd.Flags().BoolVarP(&statusQuiet, "quiet", "q", false, "Only display VM IDs")
	statusCmd.Flags().BoolVar(&statusJSON, "json", false, "Output as JSON")
	statusCmd.Flags().StringVar(&statusFormat, "format", "", "Output format (json) [deprecated: use --json]")
	_ = statusCmd.Flags().MarkDeprecated("format", "use --json instead")
	statusCmd.Flags().IntVar(&statusLimit, "limit", 50, "Maximum number of VMs to return (0 = unbounded)")
	statusCmd.Flags().IntVar(&statusOffset, "offset", 0, "Number of VMs to skip (for paging)")
}
