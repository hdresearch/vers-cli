package cmd

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/hdresearch/vers-cli/internal/auth"
	"github.com/hdresearch/vers-cli/internal/output"
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

		// Setup phase
		setup := output.New()

		// Determine VM identifier to use
		var identifier string
		if len(args) == 0 {
			var err error
			identifier, err = utils.GetCurrentHeadVM()
			if err != nil {
				return fmt.Errorf(s.NoData.Render("no VM ID provided and %w"), err)
			}
			setup.WriteStyledLine(s.HeadStatus, "Using current HEAD VM: "+identifier)
		} else {
			identifier = args[0]
		}

		setup.WriteStyledLine(s.NoData, "Fetching VM information...").
			Print()

		vm, nodeIP, err := utils.GetVmAndNodeIP(apiCtx, client, identifier)
		if err != nil {
			return fmt.Errorf(s.NoData.Render("failed to get VM information: %w"), err)
		}

		// If no node IP in headers, use default vers URL
		versHost := nodeIP
		if strings.TrimSpace(versHost) == "" {
			versUrl, err := auth.GetVersUrl()
			if err != nil {
				return err
			}
			versHost = versUrl.Hostname()
		}

		// Create VMInfo from the response
		vmInfo := utils.CreateVMInfoFromGetResponse(vm)

		if vm.State != "Running" {
			return fmt.Errorf(s.NoData.Render("VM is not running (current state: %s)"), vm.State)
		}

		if vm.NetworkInfo.SSHPort == 0 {
			return fmt.Errorf("%s", s.NoData.Render("VM does not have SSH port information available"))
		}

		// Connection status
		connection := output.New()
		connection.WriteStyledLine(s.HeadStatus, "Connecting to VM "+vmInfo.DisplayName+"...").
			Print()

		keyPath, err := auth.GetOrCreateSSHKey(vmInfo.ID, client, apiCtx)
		if err != nil {
			return fmt.Errorf("failed to get or create SSH key: %w", err)
		}

		// If we're connecting to a local machine, then use a connection string with local VM IPs. Else, use the public (DNAT'd) connection string
		sshHost := versHost
		sshPort := fmt.Sprintf("%d", vm.NetworkInfo.SSHPort)
		if utils.IsHostLocal(versHost) {
			sshHost = vm.IPAddress
			sshPort = "22"
		}

		// Debug connection info
		debug := output.New()
		debug.WriteStyledLinef(s.HeadStatus, "Connecting to %s on port %s", sshHost, sshPort).
			Print()

		sshCmd := exec.Command("ssh",
			fmt.Sprintf("root@%s", sshHost),
			"-p", sshPort,
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
