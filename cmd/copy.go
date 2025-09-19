package cmd

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/hdresearch/vers-cli/internal/auth"
	sshutil "github.com/hdresearch/vers-cli/internal/ssh"
	"github.com/hdresearch/vers-cli/internal/utils"
	"github.com/hdresearch/vers-cli/styles"
	"github.com/spf13/cobra"
)

// copyCmd represents the copy command
var copyCmd = &cobra.Command{
	Use:   "copy [vm-id|alias] <source> <destination>",
	Short: "Copy files to/from a VM using SCP",
	Long: `Copy files between your local machine and a Vers VM using SCP.
	
Examples:
  vers copy vm-123 ./local-file.txt /remote/path/
  vers copy vm-123 /remote/path/file.txt ./local-file.txt
  vers copy ./local-file.txt /remote/path/  (uses HEAD VM)
  vers copy -r ./local-dir/ /remote/path/  (recursive directory copy)`,
	Args: cobra.RangeArgs(2, 3),
	RunE: func(cmd *cobra.Command, args []string) error {
		var vmIdentifier string
		var source, destination string
		s := styles.NewStatusStyles()

		// Initialize context
		baseCtx := context.Background()
		apiCtx, cancel := context.WithTimeout(baseCtx, 30*time.Second)
		defer cancel()

		// Parse arguments based on count
		if len(args) == 2 {
			// No VM specified, use HEAD
			headVMID, err := utils.GetCurrentHeadVM()
			if err != nil {
				return fmt.Errorf(s.NoData.Render("no VM ID provided and %w"), err)
			}
			vmIdentifier = headVMID
			source = args[0]
			destination = args[1]
			fmt.Printf(s.HeadStatus.Render("Using current HEAD VM: "+vmIdentifier) + "\n")
		} else {
			// VM specified
			vmIdentifier = args[0]
			source = args[1]
			destination = args[2]
		}

		fmt.Println(s.NoData.Render("Fetching VM information..."))

		// Get VM and node information
		vm, nodeIP, err := utils.GetVmAndNodeIP(apiCtx, client, vmIdentifier)
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

		// If no node IP in headers, use default vers URL
		versHost := nodeIP
		if strings.TrimSpace(versHost) == "" {
			versUrl, err := auth.GetVersUrl()
			if err != nil {
				return err
			}
			versHost = versUrl.Hostname()
		}

		// Get SSH key
		keyPath, err := auth.GetOrCreateSSHKey(vmInfo.ID, client, apiCtx)
		if err != nil {
			return fmt.Errorf("failed to get or create SSH key: %w", err)
		}

		// Check if recursive flag is set
		recursive, err := cmd.Flags().GetBool("recursive")
		if err != nil {
			return fmt.Errorf("failed to get recursive flag: %w", err)
		}

		// Determine SSH connection details
		sshHost := versHost
		sshPort := fmt.Sprintf("%d", vm.NetworkInfo.SSHPort)
		if utils.IsHostLocal(versHost) {
			sshHost = vm.IPAddress
			sshPort = "22"
		}

		// Prepare SCP command
		scpTarget := fmt.Sprintf("root@%s", sshHost)

		// Determine if we're uploading or downloading
		var scpSource, scpDest string
		if strings.HasPrefix(source, "/") && !strings.HasPrefix(destination, "/") {
			// Downloading from remote to local
			scpSource = fmt.Sprintf("%s:%s", scpTarget, source)
			scpDest = destination
			fmt.Printf(s.HeadStatus.Render("Downloading %s from VM %s to %s\n"), source, vmInfo.DisplayName, destination)
		} else if !strings.HasPrefix(source, "/") && strings.HasPrefix(destination, "/") {
			// Uploading from local to remote
			scpSource = source
			scpDest = fmt.Sprintf("%s:%s", scpTarget, destination)
			fmt.Printf(s.HeadStatus.Render("Uploading %s to VM %s at %s\n"), source, vmInfo.DisplayName, destination)
		} else {
			// Auto-detect based on file existence
			if _, err := os.Stat(source); err == nil {
				// Local file exists, upload
				scpSource = source
				scpDest = fmt.Sprintf("%s:%s", scpTarget, destination)
				fmt.Printf(s.HeadStatus.Render("Uploading %s to VM %s at %s\n"), source, vmInfo.DisplayName, destination)
			} else {
				// Assume remote file, download
				scpSource = fmt.Sprintf("%s:%s", scpTarget, source)
				scpDest = destination
				fmt.Printf(s.HeadStatus.Render("Downloading %s from VM %s to %s\n"), source, vmInfo.DisplayName, destination)
			}
		}

		// Create the SCP command
		scpArgs := sshutil.SCPArgs(sshPort, keyPath, recursive)

		// Add source and destination
		scpArgs = append(scpArgs, scpSource, scpDest)

		scpCmd := exec.Command("scp", scpArgs...)

		// Connect command output to current terminal
		scpCmd.Stdout = os.Stdout
		scpCmd.Stderr = os.Stderr

		// Execute the command
		err = scpCmd.Run()
		if err != nil {
			if exitErr, ok := err.(*exec.ExitError); ok {
				return fmt.Errorf(s.NoData.Render("scp command exited with code %d"), exitErr.ExitCode())
			}
			return fmt.Errorf(s.NoData.Render("failed to run SCP command: %w"), err)
		}

		fmt.Printf(s.HeadStatus.Render("File copy completed successfully\n"))
		return nil
	},
}

func init() {
	rootCmd.AddCommand(copyCmd)
	copyCmd.Flags().BoolP("recursive", "r", false, "Recursively copy directories")
}
