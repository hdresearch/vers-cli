package cmd

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"time"

	vers "github.com/hdresearch/vers-sdk-go"
	"github.com/spf13/cobra"
)

// connectCmd represents the connect command
var connectCmd = &cobra.Command{
	Use:   "connect [vm-id]",
	Short: "Connect to a VM via SSH",
	Long:  `Connect to a running Vers VM via SSH. If no VM ID is provided, uses the current HEAD.`,
	Args:  cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		var vmID string
		s := NewStatusStyles()

		// If no VM ID provided, try to use the current HEAD
		if len(args) == 0 {
			var err error
			vmID, err = getCurrentHeadVM()
			if err != nil {
				return fmt.Errorf(s.NoData.Render("no VM ID provided and %v"), err)
			}
			fmt.Printf(s.HeadStatus.Render("Using current HEAD VM: "+vmID) + "\n")
		} else {
			vmID = args[0]
		}

		// Initialize SDK client and context
		baseCtx := context.Background()
		client := vers.NewClient()
		apiCtx, cancel := context.WithTimeout(baseCtx, 30*time.Second)
		defer cancel()

		// Get VM details
		fmt.Println(s.NoData.Render("Fetching VM information..."))
		vm, err := client.API.Vm.Get(apiCtx, vmID)
		if err != nil {
			return fmt.Errorf(s.NoData.Render("failed to get VM information: %v"), err)
		}

		// Check if VM is running
		if vm.State != "Running" {
			return fmt.Errorf(s.NoData.Render("VM is not running (current state: %s)"), vm.State)
		}

		// Check if we have SSH port information
		if vm.NetworkInfo.SSHPort == 0 {
			return fmt.Errorf(s.NoData.Render("VM does not have SSH port information available"))
		}

		// Check if id_rsa exists in current directory
		keyPath := "id_rsa"
		if _, err := os.Stat(keyPath); os.IsNotExist(err) {
			return fmt.Errorf(s.NoData.Render("SSH key 'id_rsa' not found in current directory"))
		}

		// Ensure key has correct permissions
		if err := os.Chmod(keyPath, 0600); err != nil {
			return fmt.Errorf(s.NoData.Render("failed to set permissions on SSH key: %v"), err)
		}

		// Build SSH command
		sshCmd := exec.Command("ssh",
			fmt.Sprintf("root@%s", "13.219.19.157"),
			"-p", fmt.Sprintf("%d", vm.NetworkInfo.SSHPort),
			"-i", keyPath,
			"-o", "StrictHostKeyChecking=no",
		)

		fmt.Println(sshCmd)

		// Set up command to use current terminal
		sshCmd.Stdin = os.Stdin
		sshCmd.Stdout = os.Stdout
		sshCmd.Stderr = os.Stderr

		fmt.Printf(s.HeadStatus.Render("Connecting to VM %s...\n"), 
			vmID)

		// Execute SSH command
		if err := sshCmd.Run(); err != nil {
			if exitErr, ok := err.(*exec.ExitError); ok {
				// SSH process exited with non-zero status
				return fmt.Errorf(s.NoData.Render("SSH connection terminated with status %d"), exitErr.ExitCode())
			}
			return fmt.Errorf(s.NoData.Render("failed to establish SSH connection: %v"), err)
		}

		return nil
	},
}

func init() {
	rootCmd.AddCommand(connectCmd)
} 