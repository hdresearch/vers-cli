package presenters

import (
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/lipgloss"
	"github.com/hdresearch/vers-cli/internal/app"
	histSvc "github.com/hdresearch/vers-cli/internal/services/history"
	"github.com/hdresearch/vers-cli/styles"
)

// Styles extracted from previous cmd/history.go implementation
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

type HistoryView struct {
	VMName  string
	VMID    string
	Commits []histSvc.CommitEntry
}

func RenderHistory(a *app.App, v HistoryView) {
	s := NewLogStyles()
	header := fmt.Sprintf("Commit History for VM: %s", v.VMName)
	fmt.Printf("\n%s\n\n", s.Header.Render(header))

	if len(v.Commits) == 0 {
		fmt.Println(s.NoData.Render("No commits found for this VM."))
		fmt.Println(s.NoData.Render("Run 'vers commit' to create your first commit."))
		return
	}

	for i, c := range v.Commits {
		ts := time.Unix(c.Timestamp, 0).Format("Mon Jan 2 15:04:05 2006 -0700")
		fmt.Printf("%s %s\n", s.CommitID.Render("Commit:"), c.ID)
		if len(c.Tags) > 0 {
			var nonEmpty []string
			for _, t := range c.Tags {
				if t = strings.TrimSpace(t); t != "" {
					nonEmpty = append(nonEmpty, t)
				}
			}
			if len(nonEmpty) > 0 {
				fmt.Printf("%s\n", s.Tag.Render(strings.Join(nonEmpty, ", ")))
			}
		}
		if c.Alias != "" && c.Alias != c.VMID {
			fmt.Printf("%s %s\n", s.Alias.Render("Alias:"), c.Alias)
		}
		fmt.Printf("%s %s\n", s.Author.Render("Author:"), c.Author)
		fmt.Printf("%s %s\n", s.Date.Render("Date:"), ts)
		fmt.Printf("%s %s\n", s.VMID.Render("VM:"), c.VMID)
		if c.HostArchitecture != "" {
			fmt.Printf("%s %s\n", s.Alias.Render("Architecture:"), c.HostArchitecture)
		}
		msg := c.Message
		if strings.TrimSpace(msg) == "" {
			msg = "(no commit message)"
		}
		fmt.Printf("\n    %s\n", s.CommitMsg.Render(msg))
		if i < len(v.Commits)-1 {
			fmt.Printf("\n%s\n\n", s.Divider.String())
		}
	}
}
