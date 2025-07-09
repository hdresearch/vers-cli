package cmd

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/hdresearch/vers-cli/internal/auth"
	"github.com/hdresearch/vers-cli/internal/utils"
	"github.com/spf13/cobra"
)

var individualTags []string
var commaSeparatedTags string

// CommitWithTagsOptions represents the request body for commits with optional tags and cluster info
type CommitWithTagsOptions struct {
	Tags      []string `json:"tags,omitempty"`
	ClusterID string   `json:"cluster_id,omitempty"`
}

// APIVmCommitResponse represents the response from the commit API
type APIVmCommitResponse struct {
	Data struct {
		ID string `json:"id"`
		// Add other fields as needed based on your actual API response
	} `json:"data"`
}

// CommitWithTags makes a commit request with optional tags array and cluster ID
func CommitWithTags(ctx context.Context, vmID string, tags []string, clusterID string) (*APIVmCommitResponse, error) {
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

	// Prepare request body if we have tags or cluster ID
	var requestBody []byte
	if len(tags) > 0 || clusterID != "" {
		// Filter out empty tags
		filteredTags := make([]string, 0)
		for _, tag := range tags {
			if trimmed := strings.TrimSpace(tag); trimmed != "" {
				filteredTags = append(filteredTags, trimmed)
			}
		}

		payload := CommitWithTagsOptions{
			Tags:      filteredTags,
			ClusterID: clusterID,
		}
		var err error
		requestBody, err = json.Marshal(payload)
		if err != nil {
			return nil, fmt.Errorf("error preparing request body: %w", err)
		}
	}

	// Create the HTTP request
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

	// Set auth header
	req.Header.Set("Authorization", "Bearer "+apiKey)

	// Execute the request
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

// combineAllTags combines tags from both --tag and --tags flags
func combineAllTags() []string {
	var allTags []string

	// Add individual tags from --tag flags
	allTags = append(allTags, individualTags...)

	// Add comma-separated tags from --tags flag
	if commaSeparatedTags != "" {
		commaTags := strings.Split(commaSeparatedTags, ",")
		for _, tag := range commaTags {
			if trimmed := strings.TrimSpace(tag); trimmed != "" {
				allTags = append(allTags, trimmed)
			}
		}
	}

	return allTags
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

		// Combine all tags from both flag types
		allTags := combineAllTags()

		fmt.Printf("Creating commit for VM '%s'\n", vmID)
		if len(allTags) > 0 {
			fmt.Printf("Tags: %s\n", strings.Join(allTags, ", "))
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

		response, err := CommitWithTags(apiCtx, vmInfo.ID, allTags, vmInfo.ClusterID)
		if err != nil {
			return fmt.Errorf("failed to commit VM '%s': %w", vmInfo.DisplayName, err)
		}

		fmt.Printf("Successfully committed VM '%s'\n", vmInfo.DisplayName)
		fmt.Printf("Commit ID: %s\n", response.Data.ID)
		if len(allTags) > 0 {
			fmt.Printf("Tags: %s\n", strings.Join(allTags, ", "))
		}

		return nil
	},
}

func init() {
	rootCmd.AddCommand(commitCmd)

	// Define flags for the commit command
	commitCmd.Flags().StringSliceVarP(&individualTags, "tag", "t", []string{}, "Individual tag for this commit (can be repeated)")
	commitCmd.Flags().StringVar(&commaSeparatedTags, "tags", "", "Comma-separated tags for this commit")
}
