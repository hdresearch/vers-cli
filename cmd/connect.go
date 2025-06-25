package cmd

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"time"

	"github.com/hdresearch/vers-cli/internal/auth"
	"github.com/hdresearch/vers-cli/internal/utils"
	"github.com/hdresearch/vers-cli/styles"
	"github.com/spf13/cobra"
)

// connectCmd represents the connect command
var connectCmd = &cobra.Command{
	Use:   "connect [vm-id|alias]",
	Short: "Connect to a VM via SSH",
	Long:  `Connect to a running Vers VM via SSH. If no VM ID or alias is provided, uses the current HEAD.`,
	Args:  cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		s := styles.NewStatusStyles()

		// Initialize context
		baseCtx := context.Background()
		apiCtx, cancel := context.WithTimeout(baseCtx, 30*time.Second)
		defer cancel()

		// Determine VM identifier to use - OPTIMIZED: single API call approach
		var identifier string
		if len(args) == 0 {
			// Use HEAD VM - get ID first (no API call)
			headVMID, err := utils.GetCurrentHeadVM()
			if err != nil {
				return fmt.Errorf(s.NoData.Render("no VM ID provided and %w"), err)
			}
			identifier = headVMID
			fmt.Printf(s.HeadStatus.Render("Using current HEAD VM: "+identifier) + "\n")
		} else {
			// Use provided identifier (could be ID or alias)
			identifier = args[0]
		}

		fmt.Println(s.NoData.Render("Fetching VM information..."))

		// Single API call gets VM data AND network info - OPTIMIZED!
		vm, nodeIP, err := utils.GetVmAndNodeIP(apiCtx, client, identifier)
		if err != nil {
			return fmt.Errorf(s.NoData.Render("failed to get VM information: %w"), err)
		}

		// Create VMInfo from the response
		vmInfo := utils.CreateVMInfoFromGetResponse(vm)

		if vm.State != "Running" {
			return fmt.Errorf(s.NoData.Render("VM is not running (current state: %s)"), vm.State)
		}

		if vm.NetworkInfo.SSHPort == 0 {
			return fmt.Errorf("%s", s.NoData.Render("VM does not have SSH port information available"))
		}

		fmt.Printf(s.HeadStatus.Render("Connecting to VM %s..."), vmInfo.DisplayName)

		// Debug info about connection
		fmt.Printf(s.HeadStatus.Render("Connecting to %s on port %d\n"), nodeIP, vm.NetworkInfo.SSHPort)

		// Use the VM ID for SSH key management
		keyPath, err := auth.GetOrCreateSSHKey(vmInfo.ID, client, apiCtx)
		if err != nil {
			return fmt.Errorf("failed to get or create SSH key: %w", err)
		}

		sshCmd := exec.Command("ssh",
			fmt.Sprintf("root@%s", nodeIP),
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
				return fmt.Errorf(s.NoData.Render("failed to run SSH command: %w"), err)
			}
		}

		return nil
	},
}

func init() {
	rootCmd.AddCommand(connectCmd)
	connectCmd.Flags().String("host", "", "Specify the host IP to connect to (overrides default)")
}
