package cmd

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

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
		versDir := ".vers"
		headFile := filepath.Join(versDir, "HEAD")

		// Check if .vers directory exists
		if _, err := os.Stat(versDir); os.IsNotExist(err) {
			return fmt.Errorf(".vers directory not found. Run 'vers init' first")
		}

		// If no arguments provided, show current HEAD
		if len(args) == 0 {
			return showCurrentHead(headFile)
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

		// Update HEAD to point to the VM
		if err := os.WriteFile(headFile, []byte(target+"\n"), 0644); err != nil {
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

// showCurrentHead displays the current HEAD information
func showCurrentHead(headFile string) error {
	headData, err := os.ReadFile(headFile)
	if err != nil {
		return fmt.Errorf("error reading HEAD: %w", err)
	}

	headContent := strings.TrimSpace(string(headData))
	if headContent == "" {
		fmt.Println("HEAD is empty. Create a VM first with 'vers run'")
		return nil
	}

	// Try to get VM details to show more information
	baseCtx := context.Background()
	apiCtx, cancel := context.WithTimeout(baseCtx, 10*time.Second)
	defer cancel()

	response, err := client.API.Vm.Get(apiCtx, headContent)
	if err != nil {
		fmt.Printf("Current HEAD: %s (unable to verify: %v)\n", headContent, err)
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
