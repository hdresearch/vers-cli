package cmd

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/lipgloss"
	"github.com/hdresearch/vers-cli/internal/output"
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

// logCommitEntry represents commit information from the API
type logCommitEntry struct {
	ID               string   `json:"ID"`
	Message          string   `json:"Message"`
	Timestamp        int64    `json:"Timestamp"`
	Tags             []string `json:"Tags"`
	Author           string   `json:"Author"`
	VMID             string   `json:"VMID"`
	Alias            string   `json:"Alias"`
	ClusterID        string   `json:"ClusterID"`
	HostArchitecture string   `json:"HostArchitecture"`
}

// commitResponse represents the API response structure
type commitResponse struct {
	Commits []logCommitEntry `json:"commits"`
}

// logCmd represents the log command
var logCmd = &cobra.Command{
	Use:   "history [vm-id|alias]",
	Short: "Display commit history",
	Long:  `Shows the commit history for the current VM or a specified VM ID or alias.`,
	Args:  cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		var vmID string
		var vmInfo *utils.VMInfo
		s := NewLogStyles()

		// Initialize SDK client and context
		baseCtx := context.Background()
		apiCtx, cancel := context.WithTimeout(baseCtx, 30*time.Second)
		defer cancel()

		// Setup output
		setup := output.New()

		// Determine VM ID to use
		if len(args) == 0 {
			// Use HEAD VM
			var err error
			vmID, err = utils.GetCurrentHeadVM()
			if err != nil {
				return fmt.Errorf("no VM ID provided and %w", err)
			}
			setup.WriteLinef("Showing commit history for current HEAD VM: %s", vmID)
		} else {
			// Use provided identifier
			vmInfo, err := utils.ResolveVMIdentifier(apiCtx, client, args[0])
			if err != nil {
				return fmt.Errorf("failed to find VM: %w", err)
			}
			vmID = vmInfo.ID
		}

		setup.WriteStyledLine(s.NoData, "Fetching VM information...").
			Print()

		if vmInfo == nil {
			response, err := client.API.Vm.Get(apiCtx, vmID)
			if err != nil {
				return fmt.Errorf("failed to get VM information: %w", err)
			}
			vmInfo = utils.CreateVMInfoFromGetResponse(response.Data)
		}

		// Fetch status
		fetch := output.New()
		fetch.WriteStyledLine(s.NoData, "Fetching commit history from server...").
			Print()

		// Make API call to get commits for this VM
		var commitResp commitResponse
		err := client.Get(apiCtx, fmt.Sprintf("/api/vm/%s/commits", vmID), nil, &commitResp)
		if err != nil {
			// If API call fails, show a helpful message
			noHistory := output.New()
			noHistory.WriteStyledLine(s.NoData, "No commit history found for this VM.").
				WriteStyledLine(s.NoData, "Commits will appear here after you run 'vers commit'.").
				Print()
			return nil
		}

		commits := commitResp.Commits

		// If no commits found, show helpful message
		if len(commits) == 0 {
			noCommits := output.New()
			noCommits.WriteLinef("\n%s\n", s.Header.Render(fmt.Sprintf("Commit History for VM: %s", vmInfo.DisplayName))).
				WriteStyledLine(s.NoData, "No commits found for this VM.").
				WriteStyledLine(s.NoData, "Run 'vers commit' to create your first commit.").
				Print()
			return nil
		}

		// Build complete commit history
		history := output.New()
		history.WriteLinef("\n%s\n", s.Header.Render(fmt.Sprintf("Commit History for VM: %s", vmInfo.DisplayName)))

		// Display commit history
		for i, commit := range commits {
			timestamp := time.Unix(commit.Timestamp, 0).Format("Mon Jan 2 15:04:05 2006 -0700")

			// Build commit info block
			history.WriteStyled(s.CommitID, "Commit: ").
				WriteStyledLine(s.CommitID, commit.ID)

			// Display tags if they exist
			if len(commit.Tags) > 0 {
				// Filter out empty tags and join with commas
				var nonEmptyTags []string
				for _, tag := range commit.Tags {
					if strings.TrimSpace(tag) != "" {
						nonEmptyTags = append(nonEmptyTags, strings.TrimSpace(tag))
					}
				}
				if len(nonEmptyTags) > 0 {
					tagsStr := strings.Join(nonEmptyTags, ", ")
					history.WriteStyledLine(s.Tag, tagsStr)
				}
			}

			// Add other commit details
			if commit.Alias != "" && commit.Alias != commit.VMID {
				history.WriteStyled(s.Alias, "Alias: ").
					WriteStyledLine(s.Alias, commit.Alias)
			}

			history.WriteStyled(s.Author, "Author: ").
				WriteStyledLine(s.Author, commit.Author).
				WriteStyled(s.Date, "Date: ").
				WriteStyledLine(s.Date, timestamp).
				WriteStyled(s.VMID, "VM: ").
				WriteStyledLine(s.VMID, commit.VMID)

			if commit.ClusterID != "" {
				history.WriteStyled(s.Alias, "Cluster: ").
					WriteStyledLine(s.Alias, commit.ClusterID)
			}
			if commit.HostArchitecture != "" {
				history.WriteStyled(s.Alias, "Architecture: ").
					WriteStyledLine(s.Alias, commit.HostArchitecture)
			}

			message := commit.Message
			if message == "" {
				message = "(no commit message)"
			}
			history.WriteLinef("\n    %s", s.CommitMsg.Render(message))

			// Add divider between commits
			if i < len(commits)-1 {
				history.WriteLinef("\n%s\n", s.Divider.String())
			}
		}

		history.Print()
		return nil
	},
}

func init() {
	rootCmd.AddCommand(logCmd)
}
