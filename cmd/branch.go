package cmd

import (
	"context"

	"github.com/hdresearch/vers-cli/internal/handlers"
	pres "github.com/hdresearch/vers-cli/internal/presenters"
	"github.com/spf13/cobra"
)

var (
	alias        string
	branchCount  int
	branchFormat string
	branchWait   bool
)

// branchCmd represents the branch command
var branchCmd = &cobra.Command{
	Use:   "branch [vm-id|alias]",
	Short: "Create a new VM from an existing VM",
	Long: `Create a new VM (branch) from the state of an existing VM. If no VM ID or alias is provided, uses the current HEAD.

Use --format json for machine-readable output.
Use --wait to block until new VMs are running.`,
	Args: cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		checkoutFlag, _ := cmd.Flags().GetBool("checkout")

		apiCtx, cancel := context.WithTimeout(context.Background(), application.Timeouts.APILong)
		defer cancel()

		target := ""
		if len(args) > 0 {
			target = args[0]
		}
		res, err := handlers.HandleBranch(apiCtx, application, handlers.BranchReq{
			Target:   target,
			Alias:    alias,
			Checkout: checkoutFlag,
			Count:    branchCount,
			Wait:     branchWait,
		})
		if err != nil {
			return err
		}

		format := pres.ParseFormat(false, branchFormat)
		switch format {
		case pres.FormatJSON:
			pres.PrintJSON(res)
		default:
			pres.RenderBranch(application, res)
		}
		return nil
	},
}

func init() {
	rootCmd.AddCommand(branchCmd)

	branchCmd.Flags().StringVarP(&alias, "alias", "n", "", "Alias for the new VM")
	branchCmd.Flags().BoolP("checkout", "c", false, "Switch to the new VM after creation")
	branchCmd.Flags().IntVar(&branchCount, "count", 1, "Number of branches to create")
	branchCmd.Flags().StringVar(&branchFormat, "format", "", "Output format (json)")
	branchCmd.Flags().BoolVar(&branchWait, "wait", false, "Wait until new VMs are running")
}
