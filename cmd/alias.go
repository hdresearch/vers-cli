package cmd

import (
	"fmt"

	"github.com/hdresearch/vers-cli/internal/utils"
	"github.com/spf13/cobra"
)

var aliasCmd = &cobra.Command{
	Use:   "alias [name]",
	Short: "Show VM ID for an alias, or list all aliases",
	Long: `Look up the VM ID for a given alias, or list all aliases if no argument is provided.

Examples:
  vers alias myvm      # Show VM ID for alias 'myvm'
  vers alias           # List all aliases`,
	Args: cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		if len(args) == 0 {
			return listAliases()
		}
		return showAlias(args[0])
	},
}

func listAliases() error {
	aliases, err := utils.LoadAliases()
	if err != nil {
		return fmt.Errorf("failed to load aliases: %w", err)
	}

	if len(aliases) == 0 {
		fmt.Println("No aliases defined.")
		return nil
	}

	for alias, vmID := range aliases {
		fmt.Printf("%s -> %s\n", alias, vmID)
	}
	return nil
}

func showAlias(name string) error {
	aliases, err := utils.LoadAliases()
	if err != nil {
		return fmt.Errorf("failed to load aliases: %w", err)
	}

	vmID, ok := aliases[name]
	if !ok {
		return fmt.Errorf("alias '%s' not found", name)
	}

	fmt.Println(vmID)
	return nil
}

func init() {
	rootCmd.AddCommand(aliasCmd)
}
