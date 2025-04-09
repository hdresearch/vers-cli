package cmd

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"
)

// checkoutCmd represents the checkout command
var checkoutCmd = &cobra.Command{
	Use:   "checkout [branch-name|vm-id]",
	Short: "Switch to a different branch or VM",
	Long: `Change the current HEAD to point to a different branch or directly to a VM ID.
If --create/-c flag is used, a new branch will be created if it doesn't exist.
If no arguments are provided, lists all available branches.`,
	Args: cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		versDir := ".vers"
		headFile := filepath.Join(versDir, "HEAD")

		// Check if .vers directory exists
		if _, err := os.Stat(versDir); os.IsNotExist(err) {
			return fmt.Errorf(".vers directory not found. Run 'vers init' first")
		}

		// If no arguments provided, list available branches
		if len(args) == 0 {
			return listBranches(versDir)
		}

		// The target to checkout (branch name or VM ID)
		target := args[0]
		createFlag, _ := cmd.Flags().GetBool("create")

		// If the target looks like a VM ID (starts with "vm-"), set HEAD directly to it
		if strings.HasPrefix(target, "vm-") {
			fmt.Printf("Checking out VM: %s\n", target)

			// Write VM ID directly to HEAD file (detached HEAD state)
			if err := os.WriteFile(headFile, []byte(target+"\n"), 0644); err != nil {
				return fmt.Errorf("failed to update HEAD: %w", err)
			}

			fmt.Printf("HEAD is now in detached state at VM '%s'\n", target)
			return nil
		}

		// Otherwise, treat it as a branch name
		// First, check if the branch exists
		branchRefPath := filepath.Join(versDir, "refs", "heads", target)
		if _, err := os.Stat(branchRefPath); os.IsNotExist(err) {
			// Branch doesn't exist
			if createFlag {
				// Create new branch based on current HEAD if --create/-c flag is used
				currentVmID, err := getCurrentHeadVM()
				if err != nil {
					return fmt.Errorf("failed to get current HEAD VM: %w", err)
				}

				// Ensure the refs/heads directory exists
				refsHeadsDir := filepath.Join(versDir, "refs", "heads")
				if err := os.MkdirAll(refsHeadsDir, 0755); err != nil {
					return fmt.Errorf("failed to create refs/heads directory: %w", err)
				}

				// Create the new branch pointing to the current VM
				if err := os.WriteFile(branchRefPath, []byte(currentVmID+"\n"), 0644); err != nil {
					return fmt.Errorf("failed to create branch: %w", err)
				}

				fmt.Printf("Created new branch '%s' pointing to VM '%s'\n", target, currentVmID)
			} else {
				return fmt.Errorf("branch '%s' does not exist. Use --create/-c flag to create it", target)
			}
		}

		// Read the branch reference to get the VM ID
		refData, err := os.ReadFile(branchRefPath)
		if err != nil {
			return fmt.Errorf("error reading branch reference: %w", err)
		}
		vmID := string(bytes.TrimSpace(refData))

		// Update HEAD to point to the branch
		newRef := fmt.Sprintf("ref: refs/heads/%s\n", target)
		if err := os.WriteFile(headFile, []byte(newRef), 0644); err != nil {
			return fmt.Errorf("failed to update HEAD: %w", err)
		}

		fmt.Printf("Switched to branch '%s' (VM: %s)\n", target, vmID)
		return nil
	},
}

// listBranches lists all available branches and marks the current one
func listBranches(versDir string) error {
	// Get current branch name
	currentBranch := ""
	headFile := filepath.Join(versDir, "HEAD")
	headData, err := os.ReadFile(headFile)
	if err == nil {
		headContent := string(bytes.TrimSpace(headData))
		if strings.HasPrefix(headContent, "ref: ") {
			refPath := strings.TrimPrefix(headContent, "ref: ")
			currentBranch = strings.TrimPrefix(refPath, "refs/heads/")
		}
	}

	// List all branches in refs/heads directory
	refsHeadsDir := filepath.Join(versDir, "refs", "heads")
	if _, err := os.Stat(refsHeadsDir); os.IsNotExist(err) {
		fmt.Println("No branches found. Create a branch with 'vers checkout -c <branch-name>'.")
		return nil
	}

	// Read directory contents
	entries, err := os.ReadDir(refsHeadsDir)
	if err != nil {
		return fmt.Errorf("failed to read branches directory: %w", err)
	}

	if len(entries) == 0 {
		fmt.Println("No branches found. Create a branch with 'vers checkout -c <branch-name>'.")
		return nil
	}

	fmt.Println("Available branches:")
	for _, entry := range entries {
		if entry.IsDir() {
			continue // Skip directories
		}

		branchName := entry.Name()
		prefix := "  "
		if branchName == currentBranch {
			prefix = "* " // Mark current branch with asterisk
		}

		// Get the VM ID for this branch
		branchPath := filepath.Join(refsHeadsDir, branchName)
		branchData, err := os.ReadFile(branchPath)
		if err != nil {
			continue // Skip if we can't read the branch file
		}
		vmID := string(bytes.TrimSpace(branchData))

		fmt.Printf("%s%s -> %s\n", prefix, branchName, vmID)
	}

	return nil
}

func init() {
	rootCmd.AddCommand(checkoutCmd)

	// Add flag to create a new branch if it doesn't exist
	checkoutCmd.Flags().BoolP("create", "c", false, "Create branch if it doesn't exist")
}
