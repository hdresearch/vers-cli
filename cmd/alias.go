package cmd

import (
	"fmt"
	"sort"

	pres "github.com/hdresearch/vers-cli/internal/presenters"
	"github.com/hdresearch/vers-cli/internal/utils"
	"github.com/spf13/cobra"
)

var (
	aliasLimit  int
	aliasOffset int
)

var aliasCmd = &cobra.Command{
	Use:   "alias [name]",
	Short: "Show VM ID for an alias, or list all aliases",
	Long: `Look up the VM ID for a given alias, or list all aliases if no argument is provided.

Examples:
  vers alias myvm      # Show VM ID for alias 'myvm'
  vers alias           # List all aliases

Pagination (when listing all):
  --limit N    Cap results at N (default 50). Use 0 for unbounded.
  --offset N   Skip the first N aliases (alphabetically by name).`,
	Args: cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		if len(args) == 0 {
			return listAliases(cmd)
		}
		return showAlias(args[0])
	},
}

func listAliases(cmd *cobra.Command) error {
	aliases, err := utils.LoadAliases()
	if err != nil {
		return fmt.Errorf("failed to load aliases: %w", err)
	}

	if len(aliases) == 0 {
		fmt.Println("No aliases defined.")
		return nil
	}

	// Sort for stable pagination.
	names := make([]string, 0, len(aliases))
	for name := range aliases {
		names = append(names, name)
	}
	sort.Strings(names)

	// TODO: aliases are stored locally; if remote alias listing ever moves
	// server-side, plumb aliasLimit/aliasOffset through to the request.
	start, end, info := pres.ApplyPaging(len(names), aliasLimit, aliasOffset)
	for _, name := range names[start:end] {
		fmt.Printf("%s -> %s\n", name, aliases[name])
	}
	pres.PrintTruncationHint(cmd.ErrOrStderr(), info)
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
	aliasCmd.Flags().IntVar(&aliasLimit, "limit", 50, "Maximum number of aliases to return (0 = unbounded)")
	aliasCmd.Flags().IntVar(&aliasOffset, "offset", 0, "Number of aliases to skip (for paging)")
}
