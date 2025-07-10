package cmd

import (
	"context"
	"fmt"
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

// logCommitEntry represents commit information from the API
type logCommitEntry struct {
	ID        string   `json:"ID"`
	Message   string   `json:"Message"`
	Timestamp int64    `json:"Timestamp"`
	Tags      []string `json:"Tags"`
	Author    string   `json:"Author"`
	VMID      string   `json:"VMID"`
	Alias     string   `json:"Alias"`
	ClusterID string   `json:"ClusterID"` // Added cluster ID
}

// commitResponse represents the API response structure
type commitResponse struct {
	Commits []logCommitEntry `json:"commits"`
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
			fmt.Printf("Showing commit history for current HEAD VM: %s\n", vmID)
		} else {
			// Use provided identifier
			vmInfo, err := utils.ResolveVMIdentifier(apiCtx, client, args[0])
			if err != nil {
				return fmt.Errorf("failed to find VM: %w", err)
			}
			vmID = vmInfo.ID
		}

		// Get VM details if we don't have them
		fmt.Println(s.NoData.Render("Fetching VM information..."))
		if vmInfo == nil {
			response, err := client.API.Vm.Get(apiCtx, vmID)
			if err != nil {
				return fmt.Errorf("failed to get VM information: %w", err)
			}
			vmInfo = utils.CreateVMInfoFromGetResponse(response.Data)
		}

		// Fetch commit history from the API
		fmt.Println(s.NoData.Render("Fetching commit history from server..."))

		// Make API call to get commits for this VM
		var commitResp commitResponse
		err := client.Get(apiCtx, fmt.Sprintf("/api/vm/%s/commits", vmID), nil, &commitResp)
		if err != nil {
			// If API call fails, show a helpful message
			fmt.Println(s.NoData.Render("No commit history found for this VM."))
			fmt.Println(s.NoData.Render("Commits will appear here after you run 'vers commit'."))
			return nil
		}

		commits := commitResp.Commits

		// If no commits found, show helpful message
		if len(commits) == 0 {
			fmt.Printf("\n%s\n\n", s.Header.Render(fmt.Sprintf("Commit History for VM: %s", vmInfo.DisplayName)))
			fmt.Println(s.NoData.Render("No commits found for this VM."))
			fmt.Println(s.NoData.Render("Run 'vers commit' to create your first commit."))
			return nil
		}

		// Display the VM info header
		fmt.Printf("\n%s\n\n", s.Header.Render(fmt.Sprintf("Commit History for VM: %s", vmInfo.DisplayName)))

		// Display commit history
		for i, commit := range commits {
			// Format timestamp
			timestamp := time.Unix(commit.Timestamp, 0).Format("Mon Jan 2 15:04:05 2006 -0700")

			// Display commit info
			fmt.Printf("%s %s\n", s.CommitID.Render("Commit:"), commit.ID)

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
					fmt.Printf("%s\n", s.Tag.Render(tagsStr))
				}
			}

			if commit.Alias != "" && commit.Alias != commit.VMID {
				fmt.Printf("%s %s\n", s.Alias.Render("Alias:"), commit.Alias)
			}
			fmt.Printf("%s %s\n", s.Author.Render("Author:"), commit.Author)
			fmt.Printf("%s %s\n", s.Date.Render("Date:"), timestamp)
			fmt.Printf("%s %s\n", s.VMID.Render("VM:"), commit.VMID)
			if commit.ClusterID != "" {
				fmt.Printf("%s %s\n", s.Alias.Render("Cluster:"), commit.ClusterID)
			}

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
