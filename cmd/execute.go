package cmd

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/hdresearch/vers-cli/internal/auth"
	"github.com/hdresearch/vers-cli/internal/utils"
	"github.com/hdresearch/vers-cli/styles"
	"github.com/spf13/cobra"
)

// executeCmd represents the execute command
var executeCmd = &cobra.Command{
	Use:   "execute [vm-id|alias] [args...]",
	Short: "Run a command on a specific VM",
	Long:  `Execute a command within the Vers environment on the specified VM. If no VM ID or alias is provided, uses the current HEAD.`,
	Args:  cobra.MinimumNArgs(1), // Require at least command
	RunE: func(cmd *cobra.Command, args []string) error {
		var vmID string
		var vmInfo *utils.VMInfo
		var commandArgs []string
		var commandStr string
		s := styles.NewStatusStyles()

		// Initialize context
		baseCtx := context.Background()
		apiCtx, cancel := context.WithTimeout(baseCtx, 30*time.Second)
		defer cancel()

		// Check if first arg is a VM ID/alias or a command - OPTIMIZED approach
		if len(args) > 1 {
			// Try to resolve the first argument as a VM identifier
			possibleVMInfo, vmErr := utils.ResolveVMIdentifier(apiCtx, client, args[0])
			if vmErr == nil {
				// First arg is a valid VM identifier
				vmInfo = possibleVMInfo
				vmID = vmInfo.ID
				commandArgs = args[1:]
			} else {
				// First arg is not a valid VM, use HEAD and treat all args as command
				headVMID, err := utils.GetCurrentHeadVM()
				if err != nil {
					return fmt.Errorf(s.NoData.Render("no VM ID provided and %w"), err)
				}
				vmID = headVMID
				fmt.Printf(s.HeadStatus.Render("Using current HEAD VM: "+vmID) + "\n")
				commandArgs = args
			}
		} else {
			// Only one arg, use HEAD and treat it as command
			headVMID, err := utils.GetCurrentHeadVM()
			if err != nil {
				return fmt.Errorf(s.NoData.Render("no VM ID provided and %w"), err)
			}
			vmID = headVMID
			fmt.Printf(s.HeadStatus.Render("Using current HEAD VM: "+vmID) + "\n")
			commandArgs = args
		}

		// Join the command arguments
		commandStr = strings.Join(commandArgs, " ")

		// Get VM and node information - OPTIMIZED: single API call
		vm, nodeIP, err := utils.GetVmAndNodeIP(apiCtx, client, vmID)
		if err != nil {
			return fmt.Errorf(s.NoData.Render("failed to get VM information: %w"), err)
		}

		// Create VMInfo from VM data if we don't have it (HEAD case or invalid first arg case)
		if vmInfo == nil {
			vmInfo = utils.CreateVMInfoFromGetResponse(vm)
		}

		if vm.State != "Running" {
			return fmt.Errorf(s.NoData.Render("VM is not running (current state: %s)"), vm.State)
		}

		if vm.NetworkInfo.SSHPort == 0 {
			return fmt.Errorf("%s", s.NoData.Render("VM does not have SSH port information available"))
		}

		// Determine the path for storing the SSH key (use VM ID)
		keyPath, err := auth.GetOrCreateSSHKey(vmInfo.ID, client, apiCtx)
		if err != nil {
			return fmt.Errorf("failed to get or create SSH key: %w", err)
		}

		// Create the SSH command with the provided command string
		sshCmd := exec.Command("ssh",
			fmt.Sprintf("root@%s", nodeIP),
			"-p", fmt.Sprintf("%d", vm.NetworkInfo.SSHPort),
			"-o", "StrictHostKeyChecking=no",
			"-o", "UserKnownHostsFile=/dev/null", // Avoid host key prompts
			"-o", "IdentitiesOnly=yes", // Only use the specified identity file
			"-o", "PreferredAuthentications=publickey", // Only attempt public key authentication
			"-o", "LogLevel=ERROR", // Add this line to suppress warnings
			"-i", keyPath, // Use the persistent key file
			commandStr) // Add the command to execute

		// Connect command output to current terminal
		sshCmd.Stdout = os.Stdout
		sshCmd.Stderr = os.Stderr

		// Execute the command
		err = sshCmd.Run()
		if err != nil {
			if exitErr, ok := err.(*exec.ExitError); ok {
				// If the command ran but returned non-zero, return the exit code
				return fmt.Errorf(s.NoData.Render("command exited with code %d"), exitErr.ExitCode())
			}
			return fmt.Errorf(s.NoData.Render("failed to run SSH command: %w"), err)
		}

		return nil
	},
}

func init() {
	rootCmd.AddCommand(executeCmd)
	executeCmd.Flags().String("host", "", "Specify the host IP to connect to (overrides default)")
}
