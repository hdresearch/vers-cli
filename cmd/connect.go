package cmd

import (
	"context"
	"fmt"
	"io/ioutil"
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

		fmt.Println(s.NoData.Render("Fetching VM information..."))
		vm, err := client.API.Vm.Get(apiCtx, vmID)
		if err != nil {
			return fmt.Errorf(s.NoData.Render("failed to get VM information: %v"), err)
		}

		if vm.State != "Running" {
			return fmt.Errorf(s.NoData.Render("VM is not running (current state: %s)"), vm.State)
		}

		if vm.NetworkInfo.SSHPort == 0 {
			return fmt.Errorf("%s", s.NoData.Render("VM does not have SSH port information available"))
		}

		fmt.Printf(s.HeadStatus.Render("Connecting to VM %s..."), vmID)

		// Get SSH key using SDK
		sshKeyBytes, err := client.API.Vm.GetSSHKey(apiCtx, vmID)
		if err != nil {
			return fmt.Errorf(s.NoData.Render("failed to get SSH key: %v"), err)
		}
		
		// Create a temporary file for the SSH key
		tmpFile, err := ioutil.TempFile("", "vers-ssh-key-*")
		if err != nil {
			return fmt.Errorf(s.NoData.Render("failed to create temporary key file: %v"), err)
		}
		defer os.Remove(tmpFile.Name()) // Ensure cleanup

		// Write key to the temporary file
		if _, err := tmpFile.Write([]byte(*sshKeyBytes)); err != nil {
			tmpFile.Close()
			return fmt.Errorf(s.NoData.Render("failed to write key to temporary file: %v"), err)
		}

		if err := tmpFile.Chmod(0600); err != nil {
			tmpFile.Close()
			return fmt.Errorf(s.NoData.Render("failed to set permissions on temporary key file: %v"), err)
		}
		
		if err := tmpFile.Close(); err != nil {
			return fmt.Errorf(s.NoData.Render("failed to close temporary key file: %v"), err)
		}

		sshCmd := exec.Command("ssh",
			fmt.Sprintf("root@%s", "13.219.19.157"), // TODO: Use vm.NetworkInfo.PublicIP or similar if available
			"-p", fmt.Sprintf("%d", vm.NetworkInfo.SSHPort),
			"-o", "StrictHostKeyChecking=no",
			"-o", "UserKnownHostsFile=/dev/null", // Avoid host key prompts
			"-i", tmpFile.Name()) // Use the temporary file

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
} 