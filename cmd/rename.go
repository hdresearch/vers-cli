package cmd

import (
	"context"
	"fmt"

	"github.com/hdresearch/vers-cli/internal/handlers"
	pres "github.com/hdresearch/vers-cli/internal/presenters"
	"github.com/spf13/cobra"
)

// renameCmd represents the rename command
var renameCmd = &cobra.Command{
	Use:   "rename [vm-id|alias|cluster-id|cluster-alias] [new-alias]",
	Short: "Rename a VM or cluster",
	Long:  `Rename a VM or cluster by setting a new alias. Use -c flag for clusters. If no ID is provided, uses the current HEAD VM.`,
	Args: func(cmd *cobra.Command, args []string) error {
		// Allow 1 or 2 arguments
		if len(args) < 1 || len(args) > 2 {
			return fmt.Errorf("accepts 1 or 2 arg(s), received %d", len(args))
		}
		return nil
	},
	RunE: func(cmd *cobra.Command, args []string) error {
		apiCtx, cancel := context.WithTimeout(context.Background(), application.Timeouts.APIMedium)
		defer cancel()
		isCluster, _ := cmd.Flags().GetBool("cluster")
		var target, newAlias string
		if isCluster {
			if len(args) != 2 {
				return fmt.Errorf("accepts 1 or 2 arg(s), received %d", len(args))
			}
			target, newAlias = args[0], args[1]
		} else {
			if len(args) == 1 {
				newAlias = args[0]
			} else {
				target, newAlias = args[0], args[1]
			}
		}
		view, err := handlers.HandleRename(apiCtx, application, handlers.RenameReq{IsCluster: isCluster, Target: target, NewAlias: newAlias})
		if err != nil {
			return err
		}
		pres.RenderRename(application, view)
		return nil
	},
}

func init() {
	rootCmd.AddCommand(renameCmd)
	renameCmd.Flags().BoolP("cluster", "c", false, "Rename a cluster instead of a VM")
}
