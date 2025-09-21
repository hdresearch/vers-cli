package cmd

import (
    "github.com/hdresearch/vers-cli/internal/tui"
    "github.com/spf13/cobra"
)

var uiCmd = &cobra.Command{
    Use:   "ui",
    Short: "Launch the interactive TUI",
    RunE: func(cmd *cobra.Command, args []string) error {
        return tui.Run(application)
    },
}

func init() { rootCmd.AddCommand(uiCmd) }

