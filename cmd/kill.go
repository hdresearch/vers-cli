package cmd

import (
	"context"

	"github.com/hdresearch/vers-cli/internal/handlers"
	"github.com/spf13/cobra"
)

var skipConfirmation bool

var killCmd = &cobra.Command{
	Use:     "delete [vm-id]...",
	Aliases: []string{"kill"},
	Short:   "Delete one or more VMs",
	Long: `Delete one or more VMs by ID. If no arguments are provided, deletes the current HEAD VM.

Examples:
  vers delete                              # Delete current HEAD VM
  vers delete vm-123abc                    # Delete single VM by ID
  vers delete vm-1 vm-2 vm-3               # Delete multiple VMs
  vers kill $(vers status -q)              # Delete all VMs
  vers delete -y vm-123abc                 # Skip confirmation`,
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx, cancel := context.WithTimeout(context.Background(), application.Timeouts.APILong)
		defer cancel()
		return handlers.HandleKill(ctx, application, handlers.KillReq{
			Targets:          args,
			SkipConfirmation: skipConfirmation,
		})
	},
}

func init() {
	rootCmd.AddCommand(killCmd)
	killCmd.Flags().BoolVarP(&skipConfirmation, "yes", "y", false, "Skip confirmation prompts")
}
