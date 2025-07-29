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

		// Build initial setup output
		var setupOutput strings.Builder

		// Determine VM ID to use
		if len(args) > 0 {
			vmInfo, err := utils.ResolveVMIdentifier(apiCtx, client, args[0])
			if err != nil {
				return fmt.Errorf("failed to find VM: %w", err)
			}
			vmID = vmInfo.ID
			setupOutput.WriteString(fmt.Sprintf("Using provided VM: %s\n", vmInfo.DisplayName))
		} else {
			// Use HEAD VM
			var err error
			vmID, err = utils.GetCurrentHeadVM()
			if err != nil {
				return fmt.Errorf("failed to get current VM: %w", err)
			}
			setupOutput.WriteString(fmt.Sprintf("Using current HEAD VM: %s\n", vmID))
		}

		// Combine all tags from both flag types
		allTags := combineAllTags()

		setupOutput.WriteString(fmt.Sprintf("Creating commit for VM '%s'\n", vmID))
		if len(allTags) > 0 {
			setupOutput.WriteString(fmt.Sprintf("Tags: %s\n", strings.Join(allTags, ", ")))
		}

		// Get VM details for alias information
		setupOutput.WriteString("Creating commit...\n")

		// Print setup messages
		fmt.Print(setupOutput.String())

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

		// Build success output
		var successOutput strings.Builder
		successOutput.WriteString(fmt.Sprintf("Successfully committed VM '%s'\n", vmInfo.DisplayName))
		successOutput.WriteString(fmt.Sprintf("Commit ID: %s\n", response.Data.CommitID))
		successOutput.WriteString(fmt.Sprintf("Cluster ID: %s\n", response.Data.ClusterID))
		successOutput.WriteString(fmt.Sprintf("Host Architecture: %s\n", response.Data.HostArchitecture))
		if len(allTags) > 0 {
			successOutput.WriteString(fmt.Sprintf("Tags: %s\n", strings.Join(allTags, ", ")))
		}

		// Print final success output
		fmt.Print(successOutput.String())
		return nil
	},
}

func init() {
	rootCmd.AddCommand(commitCmd)

	// Define flags for the commit command
	commitCmd.Flags().StringSliceVarP(&individualTags, "tag", "t", []string{}, "Individual tag for this commit (can be repeated)")
	commitCmd.Flags().StringVar(&commaSeparatedTags, "tags", "", "Comma-separated tags for this commit")
}
