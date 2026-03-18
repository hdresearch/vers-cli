package cmd

import (
	"context"

	"github.com/hdresearch/vers-cli/internal/handlers"
	pres "github.com/hdresearch/vers-cli/internal/presenters"
	"github.com/spf13/cobra"
)

var (
	statusQuiet  bool
	statusFormat string
)

// statusCmd represents the status command
var statusCmd = &cobra.Command{
	Use:   "status [vm-id|alias]",
	Short: "Get status of VMs",
	Long: `Displays the status of all VMs by default. Provide a VM ID or alias as argument for VM-specific status.

Use -q/--quiet to output just VM IDs (one per line), useful for scripting:
  vers kill $(vers status -q)              # kill all VMs
  vers info $(vers status -q | head -1)    # info on first VM

Use --format json for machine-readable output.`,
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

		format := pres.ParseFormat(statusQuiet, statusFormat)
		switch format {
		case pres.FormatQuiet:
			if res.Mode == pres.StatusVM && res.VM != nil {
				pres.PrintQuiet([]string{res.VM.VmID})
			} else {
				ids := make([]string, len(res.VMs))
				for i, vm := range res.VMs {
					ids[i] = vm.VmID
				}
				pres.PrintQuiet(ids)
			}
		case pres.FormatJSON:
			if res.Mode == pres.StatusVM && res.VM != nil {
				pres.PrintJSON(res.VM)
			} else {
				pres.PrintJSON(res.VMs)
			}
		default:
			pres.RenderStatus(application, res)
		}
		return nil
	},
}

func init() {
	rootCmd.AddCommand(statusCmd)
	statusCmd.Flags().BoolVarP(&statusQuiet, "quiet", "q", false, "Only display VM IDs")
	statusCmd.Flags().StringVar(&statusFormat, "format", "", "Output format (json)")
}
