package cmd

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/hdresearch/vers-cli/internal/auth"
	"github.com/hdresearch/vers-cli/internal/utils"
	"github.com/spf13/cobra"
)

var tag string

// CommitWithTagOptions represents the request body for commits with optional tag
type CommitWithTagOptions struct {
	Tag *string `json:"tag,omitempty"`
}

// APIVmCommitResponse represents the response from the commit API
type APIVmCommitResponse struct {
	Data struct {
		ID string `json:"id"`
		// Add other fields as needed based on your actual API response
	} `json:"data"`
}

// CommitWithTag makes a commit request with an optional tag using the same auth pattern as login.go
func CommitWithTag(ctx context.Context, vmID string, tag *string) (*APIVmCommitResponse, error) {
	// Get the base URL using the same method as login.go
	baseURL, err := auth.GetVersUrl()
	if err != nil {
		return nil, fmt.Errorf("error getting API URL: %w", err)
	}

	// Build the commit URL
	commitURL := fmt.Sprintf("%s/api/vm/%s/commit", baseURL, vmID)

	// Get the API key using the same method as the SDK
	apiKey, err := auth.GetAPIKey()
	if err != nil {
		return nil, fmt.Errorf("failed to get API key: %w", err)
	}

	// Prepare request body if we have a tag
	var requestBody []byte
	if tag != nil && *tag != "" {
		payload := CommitWithTagOptions{Tag: tag}
		requestBody, err = json.Marshal(payload)
		if err != nil {
			return nil, fmt.Errorf("error preparing request body: %w", err)
		}
	}

	// Create the HTTP request using the same pattern as login.go
	var req *http.Request
	if requestBody != nil {
		req, err = http.NewRequestWithContext(ctx, "POST", commitURL, bytes.NewBuffer(requestBody))
		if err != nil {
			return nil, fmt.Errorf("error creating request: %w", err)
		}
		req.Header.Set("Content-Type", "application/json")
	} else {
		req, err = http.NewRequestWithContext(ctx, "POST", commitURL, nil)
		if err != nil {
			return nil, fmt.Errorf("error creating request: %w", err)
		}
	}

	// Set auth header using the same pattern as login.go
	req.Header.Set("Authorization", "Bearer "+apiKey)

	// Execute the request using the same pattern as login.go
	client := &http.Client{Timeout: 60 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("error executing commit request: %w", err)
	}
	defer resp.Body.Close()

	// Handle error responses
	if resp.StatusCode == 401 {
		return nil, fmt.Errorf("authentication failed - please run 'vers login' to re-authenticate")
	}

	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("commit request failed with status %d", resp.StatusCode)
	}

	// Parse the response
	var response APIVmCommitResponse
	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		return nil, fmt.Errorf("error parsing commit response: %w", err)
	}

	return &response, nil
}

// commitCmd represents the commit command
var commitCmd = &cobra.Command{
	Use:   "commit [vm-id|alias]",
	Short: "Commit the current state of the environment",
	Long:  `Save the current state of the Vers environment as a commit. If no VM ID or alias is provided, commits the current HEAD VM.`,
	Args:  cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		var vmID string
		var vmInfo *utils.VMInfo

		// Initialize the context and SDK client
		baseCtx := context.Background()
		apiCtx, cancel := context.WithTimeout(baseCtx, 60*time.Second)
		defer cancel()

		// Determine VM ID to use
		if len(args) > 0 {
			vmInfo, err := utils.ResolveVMIdentifier(apiCtx, client, args[0])
			if err != nil {
				return fmt.Errorf("failed to find VM: %w", err)
			}
			vmID = vmInfo.ID
			fmt.Printf("Using provided VM: %s\n", vmInfo.DisplayName)
		} else {
			// Use HEAD VM
			var err error
			vmID, err = utils.GetCurrentHeadVM()
			if err != nil {
				return fmt.Errorf("failed to get current VM: %w", err)
			}
			fmt.Printf("Using current HEAD VM: %s\n", vmID)
		}

		fmt.Printf("Creating commit for VM '%s'\n", vmID)
		if tag != "" {
			fmt.Printf("Tagging commit as: %s\n", tag)
		}

		// Get VM details for alias information
		fmt.Println("Creating commit...")
		if vmInfo == nil {
			vmResponse, err := client.API.Vm.Get(apiCtx, vmID)
			if err != nil {
				return fmt.Errorf("failed to get VM details: %w", err)
			}
			vmInfo = utils.CreateVMInfoFromGetResponse(vmResponse.Data)
		}

		// Call our custom CommitWithTag function instead of the SDK's Commit method
		var tagPtr *string
		if tag != "" {
			tagPtr = &tag
		}

		response, err := CommitWithTag(apiCtx, vmInfo.ID, tagPtr)
		if err != nil {
			return fmt.Errorf("failed to commit VM '%s': %w", vmInfo.DisplayName, err)
		}

		fmt.Printf("Successfully committed VM '%s'\n", vmInfo.DisplayName)
		fmt.Printf("Commit ID: %s\n", response.Data.ID)

		return nil
	},
}

func init() {
	rootCmd.AddCommand(commitCmd)

	// Define flags for the commit command
	commitCmd.Flags().StringVarP(&tag, "tag", "t", "", "Tag for this commit")
}
