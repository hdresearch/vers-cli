package cmd

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/spf13/cobra"
)

var tag string

// logCommitEntry represents commit information (shared with log.go)
type logCommitEntry struct {
	ID        string
	Message   string
	Timestamp int64
	Tag       string
	Author    string
	VMID      string
	Alias     string
}

// writeCommitToLogFile writes commit information to a JSON log file
func writeCommitToLogFile(versDir string, vmID string, commit logCommitEntry) error {
	// Read existing commits
	logFile := filepath.Join(versDir, "logs", "commits", vmID+".json")
	var commits []logCommitEntry

	// Ensure logs/commits directory exists
	commitsDir := filepath.Join(versDir, "logs", "commits")
	if err := os.MkdirAll(commitsDir, 0755); err != nil {
		return fmt.Errorf("error creating commits directory: %w", err)
	}

	// Check if log file exists and read existing commits
	if _, err := os.Stat(logFile); err == nil {
		data, err := os.ReadFile(logFile)
		if err != nil {
			return fmt.Errorf("error reading commit log: %w", err)
		}

		if err := json.Unmarshal(data, &commits); err != nil {
			return fmt.Errorf("error parsing commit log: %w", err)
		}
	}

	// Add new commit to the list
	commits = append(commits, commit)

	// Write updated commits list
	data, err := json.MarshalIndent(commits, "", "  ")
	if err != nil {
		return fmt.Errorf("error marshaling commit data: %w", err)
	}

	if err := os.WriteFile(logFile, data, 0644); err != nil {
		return fmt.Errorf("error writing commit log: %w", err)
	}

	return nil
}

// commitCmd represents the commit command
var commitCmd = &cobra.Command{
	Use:   "commit [vm-id]",
	Short: "Commit the current state of the environment",
	Long:  `Save the current state of the Vers environment as a commit. If no VM ID is provided, commits the current HEAD VM.`,
	Args:  cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		var vmID string
		var err error

		// Check if VM ID was provided as an argument
		if len(args) > 0 {
			vmID = args[0]
			fmt.Printf("Using provided VM ID: %s\n", vmID)
		} else {
			// Get the current HEAD VM ID
			vmID, err = getCurrentHeadVM()
			if err != nil {
				return fmt.Errorf("failed to get current VM: %w", err)
			}
			fmt.Printf("Using current HEAD VM: %s\n", vmID)
		}

		fmt.Printf("Creating commit for VM '%s'\n", vmID)
		if tag != "" {
			fmt.Printf("Tagging commit as: %s\n", tag)
		}

		// Initialize the context and SDK client
		baseCtx := context.Background()
		apiCtx, cancel := context.WithTimeout(baseCtx, 60*time.Second)
		defer cancel()

		// Get VM details for alias information
		vmResponse, err := client.API.Vm.Get(apiCtx, vmID)
		if err != nil {
			return fmt.Errorf("failed to get VM details: %w", err)
		}
		vm := vmResponse.Data

		// Call the SDK to commit the VM state
		fmt.Println("Creating commit...")
		response, err := client.API.Vm.Commit(apiCtx, vmID)
		if err != nil {
			return fmt.Errorf("failed to commit VM '%s': %w", vmID, err)
		}
		commitResult := response.Data

		fmt.Printf("Successfully committed VM '%s'\n", vmID)
		fmt.Printf("Commit ID: %s\n", commitResult.ID)

		// Store commit information in .vers directory
		versDir := ".vers"
		if _, err := os.Stat(versDir); !os.IsNotExist(err) {
			// Create commit info
			commitInfo := logCommitEntry{
				ID:        commitResult.ID,
				Message:   fmt.Sprintf("Commit %s", commitResult.ID),
				Timestamp: time.Now().Unix(),
				Tag:       tag,
				Author:    "user", // Could be improved to use actual user info
				VMID:      vmID,
				Alias:     vm.Alias,
			}

			// Save commit info
			if err := writeCommitToLogFile(versDir, vmID, commitInfo); err != nil {
				fmt.Printf("Warning: Failed to store commit information: %v\n", err)
			}
		}

		return nil
	},
}

func init() {
	rootCmd.AddCommand(commitCmd)

	// Define flags for the commit command
	commitCmd.Flags().StringVarP(&tag, "tag", "t", "", "Tag for this commit")
}
