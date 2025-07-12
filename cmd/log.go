package cmd

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/charmbracelet/lipgloss"
	"github.com/hdresearch/vers-cli/internal/utils"
	"github.com/hdresearch/vers-cli/styles"
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
	Alias     lipgloss.Style
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
		Alias:     styles.BaseTextStyle.Foreground(styles.TerminalPurple),
	}
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
	Use:   "log [vm-id|alias]",
	Short: "Display commit history",
	Long:  `Shows the commit history for the current VM or a specified VM ID or alias.`,
	Args:  cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		var vmID string
		var vmInfo *utils.VMInfo
		s := NewLogStyles()
		versDir := ".vers"

		// Check if .vers directory exists
		if _, err := os.Stat(versDir); os.IsNotExist(err) {
			return fmt.Errorf(".vers directory not found. Run 'vers init' first")
		}

		// Initialize SDK client and context
		baseCtx := context.Background()
		apiCtx, cancel := context.WithTimeout(baseCtx, 30*time.Second)
		defer cancel()

		// Determine VM ID to use
		if len(args) == 0 {
			// Use HEAD VM
			var err error
			vmID, err = utils.GetCurrentHeadVM()
			if err != nil {
				return fmt.Errorf("no VM ID provided and %w", err)
			}
			// Get HEAD display name for better UX
			headDisplayName, err := utils.GetCurrentHeadDisplayName()
			if err != nil {
				headDisplayName = vmID // Fallback to VM ID
			}
			fmt.Printf("Showing commit history for current HEAD VM: %s\n", headDisplayName)
		} else {
			// Use provided identifier
			vmInfo, err := utils.ResolveVMIdentifier(apiCtx, client, args[0])
			if err != nil {
				return fmt.Errorf("failed to find VM: %w", err)
			}
			vmID = vmInfo.ID
		}

		// Get VM details
		fmt.Println(s.NoData.Render("Fetching VM information..."))
		if vmInfo == nil {
			// We need to make the API call (HEAD case)
			response, err := client.API.Vm.Get(apiCtx, vmID)
			if err != nil {
				return fmt.Errorf("failed to get VM information: %w", err)
			}
			// Create VMInfo from the response
			vmInfo = utils.CreateVMInfoFromGetResponse(response.Data)
		}

		// First, try to read existing commit log (use VM ID for file operations)
		commits, err := readCommitLogFile(versDir, vmInfo.ID)
		if err != nil {
			commits = []logCommitEntry{}
			fmt.Printf("Warning: %v\n", err)
		}

		// Check if we need to add a new commit record for this VM
		foundCurrentVM := false
		for _, commit := range commits {
			if commit.VMID == vmInfo.ID {
				foundCurrentVM = true
				break
			}
		}

		// If we don't have a commit record for this VM and we successfully got VM details,
		// create a new commit info record using data from the API
		if !foundCurrentVM {
			// Create a simple message from VM display name
			message := fmt.Sprintf("VM %s", vmInfo.DisplayName)

			// Use State as additional info
			if vmInfo.State != "" {
				message = fmt.Sprintf("VM %s (%s)", vmInfo.DisplayName, vmInfo.State)
			}

			// Create commit info for this VM using API data
			commitInfo := logCommitEntry{
				ID:        fmt.Sprintf("c%s", strings.Replace(vmInfo.ID, "vm-", "", 1)),
				Message:   message,
				Timestamp: time.Now().Unix(), // Use current time as we don't have the exact commit time
				Author:    "unknown",         // No author info from API
				VMID:      vmInfo.ID,
				Alias:     vmInfo.Alias, // Use raw alias field from VMInfo
			}

			// Add to our commits list
			commits = append(commits, commitInfo)

			// Save updated commits list
			if err := writeCommitLogFile(versDir, vmInfo.ID, commits); err != nil {
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
					Alias:     "initial-vm",
				},
				{
					ID:        "c234567890abcdef",
					Message:   "Add feature X",
					Timestamp: time.Now().Add(-24 * time.Hour).Unix(),
					Tag:       "v0.1.0",
					Author:    "user@example.com",
					VMID:      "vm-parent-456",
					Alias:     "feature-x",
				},
				{
					ID:        "c345678901abcdef",
					Message:   "Fix bug in feature X",
					Timestamp: time.Now().Unix(),
					Author:    "user@example.com",
					VMID:      vmInfo.ID,
					Alias:     vmInfo.Alias, // Use raw alias from VMInfo
				},
			}
		}

		// Display the VM info header
		fmt.Printf("\n%s\n\n", s.Header.Render(fmt.Sprintf("Commit History for VM: %s", vmInfo.DisplayName)))

		// Display commit history
		for i, commit := range commits {
			// Format timestamp
			timestamp := time.Unix(commit.Timestamp, 0).Format("Mon Jan 2 15:04:05 2006 -0700")

			// Display commit info
			fmt.Printf("%s %s\n", s.CommitID.Render("Commit:"), commit.ID)
			if commit.Tag != "" {
				fmt.Printf("%s\n", s.Tag.Render(commit.Tag))
			}
			if commit.Alias != "" {
				fmt.Printf("%s %s\n", s.Alias.Render("Alias:"), commit.Alias)
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
