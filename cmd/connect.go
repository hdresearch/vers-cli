package cmd

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"time"

	"github.com/hdresearch/vers-cli/internal/auth"
	"github.com/hdresearch/vers-cli/styles"
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
		s := styles.NewStatusStyles()

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
		apiCtx, cancel := context.WithTimeout(baseCtx, 30*time.Second)
		defer cancel()

		fmt.Println(s.NoData.Render("Fetching VM information..."))
		response, err := client.API.Vm.Get(apiCtx, vmID)
		if err != nil {
			return fmt.Errorf(s.NoData.Render("failed to get VM information: %v"), err)
		}
		vm := response.Data

		if vm.State != "Running" {
			return fmt.Errorf(s.NoData.Render("VM is not running (current state: %s)"), vm.State)
		}

		if vm.NetworkInfo.SSHPort == 0 {
			return fmt.Errorf("%s", s.NoData.Render("VM does not have SSH port information available"))
		}

		fmt.Printf(s.HeadStatus.Render("Connecting to VM %s..."), vmID)

		// Determine the path for storing the SSH key
		keyPath := getSSHKeyPath(vmID)

		// Check if SSH key already exists
		keyExists := false
		if _, err := os.Stat(keyPath); err == nil {
			keyExists = true
		}

		// If key doesn't exist, fetch it and save it
		if !keyExists {
			// Create the keys directory if it doesn't exist
			keysDir := filepath.Dir(keyPath)
			if err := os.MkdirAll(keysDir, 0755); err != nil {
				return fmt.Errorf(s.NoData.Render("failed to create keys directory: %v"), err)
			}

			// Get SSH key using SDK
			response, err := client.API.Vm.GetSSHKey(apiCtx, vmID)
			if err != nil {
				return fmt.Errorf(s.NoData.Render("failed to get SSH key: %v"), err)
			}
			sshKeyBytes := response.Data

			// Write key to file
			if err := os.WriteFile(keyPath, []byte(sshKeyBytes), 0600); err != nil {
				return fmt.Errorf(s.NoData.Render("failed to write key file: %v"), err)
			}

			fmt.Printf(s.HeadStatus.Render("SSH key saved to %s\n"), keyPath)
		} else {
			fmt.Printf(s.HeadStatus.Render("Using existing SSH key from %s\n"), keyPath)
		}

		hostIP := auth.GetVersUrl()

		// Debug info about connection
		fmt.Printf(s.HeadStatus.Render("Connecting to %s on port %d\n"), hostIP, vm.NetworkInfo.SSHPort)

		keyPath, err := auth.GetOrCreateSSHKey(vmID, client, apiCtx)
		if err != nil {
			return fmt.Errorf("failed to get or create SSH key: %w", err)
		}

		sshCmd := exec.Command("ssh",
			fmt.Sprintf("root@%s", hostIP),
			"-p", fmt.Sprintf("%d", vm.NetworkInfo.SSHPort),
			"-o", "StrictHostKeyChecking=no",
			"-o", "UserKnownHostsFile=/dev/null", // Avoid host key prompts
			"-o", "IdentitiesOnly=yes", // Only use the specified identity file
			"-o", "PreferredAuthentications=publickey", // Only attempt public key authentication
			"-i", keyPath) // Use the persistent key file

		sshCmd.Stdout = os.Stdout
		sshCmd.Stderr = os.Stderr
		sshCmd.Stdin = os.Stdin // Connect terminal stdin for interactive session

		err = sshCmd.Run()

		if err != nil {
			if _, ok := err.(*exec.ExitError); !ok {
				return fmt.Errorf(s.NoData.Render("failed to run SSH command: %v"), err)
			}
		}

		return nil
	},
}

func init() {
	rootCmd.AddCommand(connectCmd)
	connectCmd.Flags().String("host", "", "Specify the host IP to connect to (overrides default)")
}
