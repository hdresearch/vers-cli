package cmd

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/exec"
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
		response, err := client.API.Vm.Get(apiCtx, vmID)
		if err != nil {
			return fmt.Errorf(s.NoData.Render("failed to get VM information: %w"), err)
		}
		vm := response.Data

		if vm.State != "Running" {
			return fmt.Errorf(s.NoData.Render("VM is not running (current state: %s)"), vm.State)
		}

		if vm.NetworkInfo.SSHPort == 0 {
			return fmt.Errorf("%s", s.NoData.Render("VM does not have SSH port information available"))
		}

		fmt.Printf(s.HeadStatus.Render("Connecting to VM %s..."), vmID)

		// Get the node's public IP from response headers (preferred)
		// Fall back to load balancer URL if header not present
		var hostIP string

		// Try to get node IP from headers using raw HTTP request
		if nodeIP, err := getNodeIPForVM(vmID); err == nil {
			hostIP = nodeIP
		} else {
			// Fallback to load balancer URL
			hostIP = auth.GetVersUrl()
			if os.Getenv("VERS_DEBUG") == "true" {
				fmt.Printf("[DEBUG] Failed to get node IP, using fallback: %v\n", err)
			}
		}

		// Debug info about connection
		fmt.Printf(s.HeadStatus.Render("Connecting to %s on port %d\n"), hostIP, vm.NetworkInfo.SSHPort)

		keyPath, err := auth.GetOrCreateSSHKey(vm.ID, client, apiCtx)
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
				return fmt.Errorf(s.NoData.Render("failed to run SSH command: %w"), err)
			}
		}

		return nil
	},
}

// getNodeIPForVM makes a raw HTTP request to get the node IP from headers
func getNodeIPForVM(vmID string) (string, error) {
	// Get API key for authentication
	apiKey, err := auth.GetAPIKey()
	if err != nil {
		return "", fmt.Errorf("failed to get API key: %w", err)
	}

	// Construct the URL using the same base URL logic
	baseURL := "https://" + auth.GetVersUrl()
	if auth.GetVersUrl() != "api.vers.sh" {
		baseURL = "http://" + auth.GetVersUrl()
	}
	url := baseURL + "/api/vm/" + vmID

	// Create HTTP request
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return "", fmt.Errorf("failed to create request: %w", err)
	}

	// Add authentication header
	req.Header.Set("Authorization", "Bearer "+apiKey)

	// Make the request
	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("failed to make request: %w", err)
	}
	defer resp.Body.Close()

	// Get the node IP from headers
	nodeIP := resp.Header.Get("X-Node-IP")
	if nodeIP != "" && nodeIP != "unknown" {
		return nodeIP, nil
	}

	return "", fmt.Errorf("no node IP found in response headers")
}

func init() {
	rootCmd.AddCommand(connectCmd)
	connectCmd.Flags().String("host", "", "Specify the host IP to connect to (overrides default)")
}
