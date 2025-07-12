package cmd

import (
	"context"
	"fmt"
	"time"

	"github.com/hdresearch/vers-cli/internal/utils"
	"github.com/spf13/cobra"
)

// checkoutCmd represents the checkout command
var checkoutCmd = &cobra.Command{
	Use:   "checkout [vm-id|alias]",
	Short: "Switch to a different VM",
	Long: `Change the current HEAD to point to a different VM by ID or alias.
If no arguments are provided, shows the current HEAD.`,
	Args: cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		// If no arguments provided, show current HEAD
		if len(args) == 0 {
			return showCurrentHead()
		}

		target := args[0]

		// Initialize context
		baseCtx := context.Background()
		apiCtx, cancel := context.WithTimeout(baseCtx, 10*time.Second)
		defer cancel()

		fmt.Printf("Verifying VM '%s'...\n", target)

		// Use utils to resolve and set HEAD (now stores both ID and alias)
		vmInfo, err := utils.SetHeadFromIdentifier(apiCtx, client, target)
		if err != nil {
			return fmt.Errorf("failed to switch to VM '%s': %w", target, err)
		}

		fmt.Printf("Switched to VM '%s' (State: %s)\n", vmInfo.DisplayName, vmInfo.State)
		return nil
	},
}

// showCurrentHead displays the current HEAD information using utils
func showCurrentHead() error {
	// Use the new function that gets display name directly from HEAD
	headDisplayName, err := utils.GetCurrentHeadDisplayName()
	if err != nil {
		return err
	}

	// If we need state info, we still need to make an API call
	baseCtx := context.Background()
	apiCtx, cancel := context.WithTimeout(baseCtx, 10*time.Second)
	defer cancel()

	vmInfo, err := utils.GetCurrentHeadVMInfo(apiCtx, client)
	if err != nil {
		return err
	}

	fmt.Printf("Current HEAD: %s (State: %s)\n", headDisplayName, vmInfo.State)
	return nil
}

func init() {
	rootCmd.AddCommand(checkoutCmd)
}
