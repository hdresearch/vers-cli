package cmd

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/hdresearch/vers-cli/internal/utils"
	"github.com/hdresearch/vers-sdk-go"
	"github.com/spf13/cobra"
)

var individualTags []string
var commaSeparatedTags string

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

		body := vers.APIVmCommitParams{
			VmCommitRequest: vers.VmCommitRequestParam{
				Tags: vers.F(allTags),
			},
		}
		response, err := client.API.Vm.Commit(apiCtx, vmInfo.ID, body)
		if err != nil {
			return fmt.Errorf("failed to commit VM '%s': %w", vmInfo.DisplayName, err)
		}

		fmt.Printf("Successfully committed VM '%s'\n", vmInfo.DisplayName)
		fmt.Printf("Commit ID: %s\n", response.Data.CommitID)
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
