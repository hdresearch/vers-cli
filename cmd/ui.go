package cmd

import (
	"fmt"

	"github.com/hdresearch/vers-cli/internal/tui"
	"github.com/hdresearch/vers-cli/styles"
	"github.com/spf13/cobra"
)

var uiCmd = &cobra.Command{
	Use:   "ui",
	Short: "Launch the interactive TUI (EXPERIMENTAL)",
	RunE: func(cmd *cobra.Command, args []string) error {
		// Big serious warning
		banner := `
!!! WARNING: Experimental Feature !!!

The Vers TUI is experimental and subject to change.
It may be unstable and is not recommended for production use.
Please report issues to the project repository.
`
		fmt.Println(styles.ErrorTextStyle.Padding(1, 0).Render(banner))
		return tui.Run(application)
	},
}

func init() {
	// TUI disabled
	// rootCmd.AddCommand(uiCmd)
}
