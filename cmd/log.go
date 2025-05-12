package cmd

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/charmbracelet/lipgloss"
	"github.com/hdresearch/vers-cli/styles"
	vers "github.com/hdresearch/vers-sdk-go"
	"github.com/spf13/cobra"
)

// LogStyles contains all styles used in the log command
type LogStyles struct {
	Container lipgloss.Style
	Header    lipgloss.Style
	CommitID  lipgloss.Style
	CommitMsg lipgloss.Style
	Author    lipgloss.Style
	Date      lipgloss.Style
	Tag       lipgloss.Style
	VMID      lipgloss.Style
	NoData    lipgloss.Style
	Divider   lipgloss.Style
	Branch    lipgloss.Style
}

// NewLogStyles initializes and returns all styles used in the log command
func NewLogStyles() LogStyles {
	containerStyle := styles.AppStyle

	return LogStyles{
		Container: containerStyle,
		Header:    styles.HeaderStyle,
		CommitID:  styles.PrimaryTextStyle.Bold(true).Foreground(styles.TerminalYellow),
		CommitMsg: styles.BaseTextStyle.Foreground(styles.TerminalWhite),
		Author:    styles.BaseTextStyle.Italic(true).Foreground(styles.TerminalGreen),
		Date:      styles.MutedTextStyle,
		Tag:       styles.BaseTextStyle.Background(styles.TerminalBlue).Foreground(styles.TerminalWhite).Padding(0, 1),
		VMID:      styles.VmIDStyle,
		NoData:    styles.MutedTextStyle.Padding(1, 0),
		Divider:   styles.MutedTextStyle.SetString("────────────────────────────────────────────────"),
		Branch:    styles.BaseTextStyle.Foreground(styles.TerminalPurple),
	}
}

// logCommitEntry represents commit information
type logCommitEntry struct {
	ID        string
	Message   string
	Timestamp int64
	Tag       string
	Author    string
	VMID      string
	Branch    string
}

// readRefsHeads reads all branches from the .vers/refs/heads directory
func readRefsHeads(versDir string) (map[string]string, error) {
	branches := make(map[string]string)
	refsHeadsDir := filepath.Join(versDir, "refs", "heads")

	// Check if directory exists
	if _, err := os.Stat(refsHeadsDir); os.IsNotExist(err) {
		return branches, nil
	}

	entries, err := os.ReadDir(refsHeadsDir)
	if err != nil {
		return nil, fmt.Errorf("failed to read branches directory: %w", err)
	}

	for _, entry := range entries {
		if entry.IsDir() {
			continue // Skip directories
		}

		branchName := entry.Name()
		branchPath := filepath.Join(refsHeadsDir, branchName)
		branchData, err := os.ReadFile(branchPath)
		if err != nil {
			continue // Skip if can't read
		}

		vmID := string(bytes.TrimSpace(branchData))
		branches[vmID] = branchName
	}

	return branches, nil
}

// readCommitLogFile reads commit information from a JSON log file
func readCommitLogFile(versDir, vmID string) ([]logCommitEntry, error) {
	logFile := filepath.Join(versDir, "logs", "commits", vmID+".json")

	// Check if log file exists
	if _, err := os.Stat(logFile); os.IsNotExist(err) {
		return nil, nil // No error, just no log file
	}

	data, err := os.ReadFile(logFile)
	if err != nil {
		return nil, fmt.Errorf("error reading commit log: %w", err)
	}

	var commits []logCommitEntry
	if err := json.Unmarshal(data, &commits); err != nil {
		return nil, fmt.Errorf("error parsing commit log: %w", err)
	}

	return commits, nil
}

// writeCommitLogFile writes commit information to a JSON log file
func writeCommitLogFile(versDir string, vmID string, commits []logCommitEntry) error {
	// Ensure logs/commits directory exists
	commitsDir := filepath.Join(versDir, "logs", "commits")
	if err := os.MkdirAll(commitsDir, 0755); err != nil {
		return fmt.Errorf("error creating commits directory: %w", err)
	}

	logFile := filepath.Join(commitsDir, vmID+".json")

	data, err := json.MarshalIndent(commits, "", "  ")
	if err != nil {
		return fmt.Errorf("error marshaling commit data: %w", err)
	}

	if err := os.WriteFile(logFile, data, 0644); err != nil {
		return fmt.Errorf("error writing commit log: %w", err)
	}

	return nil
}

// logCmd represents the log command
var logCmd = &cobra.Command{
	Use:   "log [vm-id]",
	Short: "Display commit history",
	Long:  `Shows the commit history for the current VM or a specified VM ID.`,
	Args:  cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		var vmID string
		s := NewLogStyles()
		versDir := ".vers"

		// Check if .vers directory exists
		if _, err := os.Stat(versDir); os.IsNotExist(err) {
			return fmt.Errorf(".vers directory not found. Run 'vers init' first")
		}

		// If no VM ID provided, try to use the current HEAD
		if len(args) == 0 {
			var err error
			vmID, err = getCurrentHeadVM()
			if err != nil {
				return fmt.Errorf("no VM ID provided and %w", err)
			}
			fmt.Printf("Showing commit history for current HEAD VM: %s\n", vmID)
		} else {
			vmID = args[0]
		}

		// Get branch information
		branches, err := readRefsHeads(versDir)
		if err != nil {
			fmt.Printf("Warning: Failed to read branch information: %v\n", err)
		}

		// Initialize SDK client and context
		baseCtx := context.Background()
		client := vers.NewClient()
		apiCtx, cancel := context.WithTimeout(baseCtx, 30*time.Second)
		defer cancel()

		// Get VM details to verify it exists
		fmt.Println(s.NoData.Render("Fetching VM information..."))
		response, err := client.API.Vm.Get(apiCtx, vmID)
		if err != nil {
			return fmt.Errorf("failed to get VM information: %w", err)
		}
		vm := response.Data

		// First, try to read existing commit log
		commits, err := readCommitLogFile(versDir, vmID)
		if err != nil {
			fmt.Printf("Warning: %v\n", err)
			commits = []logCommitEntry{}
		}

		// Check if we need to add a new commit record for this VM
		foundCurrentVM := false
		for _, commit := range commits {
			if commit.VMID == vmID {
				foundCurrentVM = true
				break
			}
		}

		// If we don't have a commit record for this VM and we successfully got VM details,
		// create a new commit info record using data from the API
		if !foundCurrentVM {
			// Determine branch name if known
			branchName := branches[vmID]

			// Create a simple message from VM ID if no other information is available
			message := fmt.Sprintf("VM %s", vmID)

			// Use State as additional info
			if vm.State != "" {
				message = fmt.Sprintf("VM %s (%s)", vmID, vm.State)
			}

			// Create commit info for this VM using API data
			commitInfo := logCommitEntry{
				ID:        fmt.Sprintf("c%s", strings.Replace(vmID, "vm-", "", 1)),
				Message:   message,
				Timestamp: time.Now().Unix(), // Use current time as we don't have the exact commit time
				Author:    "unknown",         // No author info from API
				VMID:      vmID,
				Branch:    branchName,
			}

			// Add to our commits list
			commits = append(commits, commitInfo)

			// Save updated commits list
			if err := writeCommitLogFile(versDir, vmID, commits); err != nil {
				fmt.Printf("Warning: Failed to update commit log: %v\n", err)
			}
		}

		// Sort commits by timestamp (newest first)
		sort.Slice(commits, func(i, j int) bool {
			return commits[i].Timestamp > commits[j].Timestamp
		})

		// If no commits found, use mock data
		if len(commits) == 0 {
			fmt.Println(s.NoData.Render("No commit history found. Using mock data for demonstration."))

			// Mock data for demonstration
			commits = []logCommitEntry{
				{
					ID:        "c123456789abcdef",
					Message:   "Initial commit",
					Timestamp: time.Now().Add(-48 * time.Hour).Unix(),
					Author:    "user@example.com",
					VMID:      "vm-ancestor-123",
				},
				{
					ID:        "c234567890abcdef",
					Message:   "Add feature X",
					Timestamp: time.Now().Add(-24 * time.Hour).Unix(),
					Tag:       "v0.1.0",
					Author:    "user@example.com",
					VMID:      "vm-parent-456",
					Branch:    "feature-x",
				},
				{
					ID:        "c345678901abcdef",
					Message:   "Fix bug in feature X",
					Timestamp: time.Now().Unix(),
					Author:    "user@example.com",
					VMID:      vmID,
					Branch:    branches[vmID],
				},
			}
		}

		// Display the VM info header
		fmt.Printf("\n%s\n\n", s.Header.Render(fmt.Sprintf("Commit History for VM: %s", vmID)))

		// Display commit history
		for i, commit := range commits {
			// Format timestamp
			timestamp := time.Unix(commit.Timestamp, 0).Format("Mon Jan 2 15:04:05 2006 -0700")

			// Display commit info
			fmt.Printf("%s %s\n", s.CommitID.Render("Commit:"), commit.ID)
			if commit.Tag != "" {
				fmt.Printf("%s\n", s.Tag.Render(commit.Tag))
			}
			if commit.Branch != "" {
				fmt.Printf("%s %s\n", s.Branch.Render("Branch:"), commit.Branch)
			}
			fmt.Printf("%s %s\n", s.Author.Render("Author:"), commit.Author)
			fmt.Printf("%s %s\n", s.Date.Render("Date:"), timestamp)
			fmt.Printf("%s %s\n", s.VMID.Render("VM:"), commit.VMID)

			message := commit.Message
			if message == "" {
				message = "(no commit message)"
			}
			fmt.Printf("\n    %s\n", s.CommitMsg.Render(message))

			// Add divider between commits
			if i < len(commits)-1 {
				fmt.Printf("\n%s\n\n", s.Divider.String())
			}
		}

		return nil
	},
}

func init() {
	rootCmd.AddCommand(logCmd)
}
