package cmd

import (
	"context"
	"fmt"
	"time"

	"github.com/hdresearch/vers-cli/internal/handlers"
	pres "github.com/hdresearch/vers-cli/internal/presenters"
	"github.com/hdresearch/vers-cli/internal/utils"
	vers "github.com/hdresearch/vers-sdk-go"
	"github.com/spf13/cobra"
)

// commitCmd is the parent command for commit operations.
// When invoked without a subcommand, it commits the current HEAD VM (backward compat).
var commitCmd = &cobra.Command{
	Use:   "commit [vm-id|alias]",
	Short: "Commit the current state of the environment",
	Long: `Save the current state of the Vers environment as a commit.
If no VM ID or alias is provided, commits the current HEAD VM.

Subcommands:
  list       List your commits
  delete     Delete a commit
  history    Show the parent commit chain
  publish    Make a commit public
  unpublish  Make a commit private`,
	Args: cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		var vmID string
		var vmInfo *utils.VMInfo

		baseCtx := context.Background()
		apiCtx, cancel := context.WithTimeout(baseCtx, 60*time.Second)
		defer cancel()

		if len(args) > 0 {
			vmInfo, err := utils.ResolveVMIdentifier(apiCtx, client, args[0])
			if err != nil {
				return fmt.Errorf("failed to find VM: %w", err)
			}
			vmID = vmInfo.ID
			fmt.Printf("Using provided VM: %s\n", vmInfo.DisplayName)
		} else {
			var err error
			vmID, err = utils.GetCurrentHeadVM()
			if err != nil {
				return fmt.Errorf("failed to get current VM: %w", err)
			}
			fmt.Printf("Using current HEAD VM: %s\n", vmID)
		}

		fmt.Printf("Creating commit for VM '%s'\n", vmID)

		fmt.Println("Creating commit...")
		if vmInfo == nil {
			vmInfo = &utils.VMInfo{
				ID:          vmID,
				DisplayName: vmID,
			}
		}

		response, err := client.Vm.Commit(apiCtx, vmInfo.ID, vers.VmCommitParams{})
		if err != nil {
			return fmt.Errorf("failed to commit VM '%s': %w", vmInfo.DisplayName, err)
		}

		fmt.Printf("Successfully committed VM '%s'\n", vmInfo.DisplayName)
		fmt.Printf("Commit ID: %s\n", response.CommitID)

		return nil
	},
}

var commitListPublic bool

var commitListCmd = &cobra.Command{
	Use:   "list",
	Short: "List your commits",
	Long:  `List all commits owned by your account. Use --public to list publicly shared commits instead.`,
	Args:  cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		apiCtx, cancel := context.WithTimeout(context.Background(), application.Timeouts.APIMedium)
		defer cancel()

		res, err := handlers.HandleCommitsList(apiCtx, application, handlers.CommitsListReq{
			Public: commitListPublic,
		})
		if err != nil {
			return err
		}
		pres.RenderCommitsList(application, res)
		return nil
	},
}

var commitDeleteCmd = &cobra.Command{
	Use:   "delete <commit-id>",
	Short: "Delete a commit",
	Long:  `Permanently delete a commit. The commit must not have any active VMs running from it.`,
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		apiCtx, cancel := context.WithTimeout(context.Background(), application.Timeouts.APIMedium)
		defer cancel()

		err := handlers.HandleCommitDelete(apiCtx, application, handlers.CommitDeleteReq{
			CommitID: args[0],
		})
		if err != nil {
			return err
		}
		fmt.Printf("✓ Commit %s deleted\n", args[0])
		return nil
	},
}

var commitHistoryCmd = &cobra.Command{
	Use:   "history <commit-id>",
	Short: "Show the parent commit chain",
	Long:  `Display the chain of parent commits leading up to a given commit.`,
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		apiCtx, cancel := context.WithTimeout(context.Background(), application.Timeouts.APIMedium)
		defer cancel()

		res, err := handlers.HandleCommitParents(apiCtx, application, handlers.CommitParentsReq{
			CommitID: args[0],
		})
		if err != nil {
			return err
		}
		pres.RenderCommitParents(application, res)
		return nil
	},
}

var commitPublishCmd = &cobra.Command{
	Use:   "publish <commit-id>",
	Short: "Make a commit public",
	Long:  `Make a commit publicly accessible so others can restore or branch from it.`,
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		apiCtx, cancel := context.WithTimeout(context.Background(), application.Timeouts.APIMedium)
		defer cancel()

		info, err := handlers.HandleCommitUpdate(apiCtx, application, handlers.CommitUpdateReq{
			CommitID: args[0],
			IsPublic: true,
		})
		if err != nil {
			return err
		}
		fmt.Printf("✓ Commit %s is now public\n", info.CommitID)
		return nil
	},
}

var commitUnpublishCmd = &cobra.Command{
	Use:   "unpublish <commit-id>",
	Short: "Make a commit private",
	Long:  `Make a commit private so only you can access it.`,
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		apiCtx, cancel := context.WithTimeout(context.Background(), application.Timeouts.APIMedium)
		defer cancel()

		info, err := handlers.HandleCommitUpdate(apiCtx, application, handlers.CommitUpdateReq{
			CommitID: args[0],
			IsPublic: false,
		})
		if err != nil {
			return err
		}
		fmt.Printf("✓ Commit %s is now private\n", info.CommitID)
		return nil
	},
}

func init() {
	rootCmd.AddCommand(commitCmd)

	commitListCmd.Flags().BoolVar(&commitListPublic, "public", false, "List public commits instead of your own")
	commitCmd.AddCommand(commitListCmd)
	commitCmd.AddCommand(commitDeleteCmd)
	commitCmd.AddCommand(commitHistoryCmd)
	commitCmd.AddCommand(commitPublishCmd)
	commitCmd.AddCommand(commitUnpublishCmd)
}
