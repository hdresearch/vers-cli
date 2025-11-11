package cmd

import (
	"context"
	"time"

	"github.com/hdresearch/vers-cli/internal/handlers"
	"github.com/spf13/cobra"
)

var (
	skipConfirmation bool
	recursive        bool
)

var killCmd = &cobra.Command{
	Use:     "delete [vm-id]...",
	Aliases: []string{"kill"},
	Short:   "Delete one or more VMs",
	Long: `Delete one or more VMs by ID. If no arguments are provided, deletes the current HEAD VM.

Examples:
  vers delete                              # Delete current HEAD VM
  vers delete vm-123abc                    # Delete single VM by ID
  vers delete vm-1 vm-2 vm-3               # Delete multiple VMs by ID
  vers delete -y                           # Delete HEAD VM without confirmation
  vers delete -r vm-with-children          # Recursively delete VM and all its children
  vers delete -y -r vm-with-children       # Skip confirmations AND delete children`,
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
		defer cancel()
		req := handlers.KillReq{
			Targets:          args,
			SkipConfirmation: skipConfirmation,
			Recursive:        recursive,
		}
		return handlers.HandleKill(ctx, application, req)
	},
}

func init() {
	rootCmd.AddCommand(killCmd)
	killCmd.Flags().BoolVarP(&skipConfirmation, "yes", "y", false, "Skip confirmation prompts")
	killCmd.Flags().BoolVarP(&recursive, "recursive", "r", false, "Recursively delete all children")
}
