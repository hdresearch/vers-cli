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

		// Verify the VM/alias exists before switching
		baseCtx := context.Background()
		apiCtx, cancel := context.WithTimeout(baseCtx, 10*time.Second)
		defer cancel()

		fmt.Printf("Verifying VM '%s'...\n", target)
		response, err := client.API.Vm.Get(apiCtx, target)
		if err != nil {
			return fmt.Errorf("failed to find VM '%s': %w", target, err)
		}
		vm := response.Data

		// Use utils to update HEAD
		if err := utils.SetHead(target); err != nil {
			return fmt.Errorf("failed to update HEAD: %w", err)
		}

		// Show success message with VM details
		displayName := vm.Alias
		if displayName == "" {
			displayName = vm.ID
		}

		fmt.Printf("Switched to VM '%s' (State: %s)\n", displayName, vm.State)
		return nil
	},
}

// showCurrentHead displays the current HEAD information using utils
func showCurrentHead() error {
	headVM, err := utils.GetCurrentHeadVM()
	if err != nil {
		return err
	}

	// Try to get VM details to show more information
	baseCtx := context.Background()
	apiCtx, cancel := context.WithTimeout(baseCtx, 10*time.Second)
	defer cancel()

	response, err := client.API.Vm.Get(apiCtx, headVM)
	if err != nil {
		fmt.Printf("Current HEAD: %s (unable to verify: %v)\n", headVM, err)
		return nil
	}

	vm := response.Data
	displayName := vm.Alias
	if displayName == "" {
		displayName = vm.ID
	}

	fmt.Printf("Current HEAD: %s (State: %s)\n", displayName, vm.State)
	return nil
}

func init() {
	rootCmd.AddCommand(checkoutCmd)
}
