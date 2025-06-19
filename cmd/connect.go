package cmd

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"time"

	"github.com/hdresearch/vers-cli/internal/auth"
	"github.com/hdresearch/vers-cli/styles"
	"github.com/hdresearch/vers-sdk-go"
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
				return fmt.Errorf(s.NoData.Render("no VM ID provided and %w"), err)
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

		vm, nodeIP, err := GetVmAndNodeIP(apiCtx, vmID)
		if err != nil {
			return fmt.Errorf(s.NoData.Render("failed to get VM information: %w"), err)
		}

		if vm.State != "Running" {
			return fmt.Errorf(s.NoData.Render("VM is not running (current state: %s)"), vm.State)
		}

		if vm.NetworkInfo.SSHPort == 0 {
			return fmt.Errorf("%s", s.NoData.Render("VM does not have SSH port information available"))
		}

		fmt.Printf(s.HeadStatus.Render("Connecting to VM %s..."), vmID)

		// Debug info about connection
		fmt.Printf(s.HeadStatus.Render("Connecting to %s on port %d\n"), nodeIP, vm.NetworkInfo.SSHPort)

		keyPath, err := auth.GetOrCreateSSHKey(vm.ID, client, apiCtx)
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

// GetVmAndNodeIP retrieves VM information and the node IP from headers in a single request
func GetVmAndNodeIP(ctx context.Context, vmID string) (vers.APIVmGetResponseData, string, error) {
	// Use the lower-level client method to get both response data AND headers
	var rawResponse *http.Response
	err := client.Get(ctx, "/api/vm/"+vmID, nil, &rawResponse)
	if err != nil {
		return vers.APIVmGetResponseData{}, "", err
	}
	defer rawResponse.Body.Close()

	// Parse the response body using the SDK types
	var response vers.APIVmGetResponse
	if err := json.NewDecoder(rawResponse.Body).Decode(&response); err != nil {
		return vers.APIVmGetResponseData{}, "", fmt.Errorf("failed to decode VM response: %w", err)
	}

	// Extract the node IP from headers
	nodeIP := rawResponse.Header.Get("X-Node-IP")

	// Use the node IP from headers or fallback to env override, and then static load balancer host
	var hostIP string
	if nodeIP != "" {
		hostIP = nodeIP
	} else {
		hostIP, err = auth.GetVersUrlHost()
		if err != nil {
			return vers.APIVmGetResponseData{}, "", fmt.Errorf("failed to get host IP: %w", err)
		}
		if os.Getenv("VERS_DEBUG") == "true" {
			fmt.Printf("[DEBUG] No node IP in headers, using fallback: %s\n", hostIP)
		}
	}

	return response.Data, hostIP, nil
}

func init() {
	rootCmd.AddCommand(connectCmd)
	connectCmd.Flags().String("host", "", "Specify the host IP to connect to (overrides default)")
}
