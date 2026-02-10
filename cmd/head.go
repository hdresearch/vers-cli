package cmd

import (
	"fmt"

	"github.com/hdresearch/vers-cli/internal/utils"
	"github.com/spf13/cobra"
)

// headCmd represents the head command
var headCmd = &cobra.Command{
	Use:   "head",
	Short: "Display the current HEAD VM ID",
	Long:  `Displays the VM ID that HEAD currently points to. Useful for programmatic use of the CLI.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		vmID, err := utils.GetCurrentHeadVM()
		if err != nil {
			return fmt.Errorf("no HEAD set: %w", err)
		}
		fmt.Println(vmID)
		return nil
	},
}

func init() {
	rootCmd.AddCommand(headCmd)
}
