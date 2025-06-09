package styles

import (
	"github.com/charmbracelet/lipgloss"
)

// BranchStyles contains all styles used in the branch command
type BranchStyles struct {
	// Base container style
	Container lipgloss.Style

	// Headers and titles
	Header     lipgloss.Style
	SubHeader  lipgloss.Style
	ListHeader lipgloss.Style

	// Branch and VM identifiers
	BranchName   lipgloss.Style
	VMID         lipgloss.Style
	CurrentState lipgloss.Style

	// Status and progress
	Progress lipgloss.Style
	Success  lipgloss.Style
	Warning  lipgloss.Style
	Error    lipgloss.Style

	// Information and help
	Info       lipgloss.Style
	InfoLabel  lipgloss.Style
	InfoValue  lipgloss.Style
	ListItem   lipgloss.Style
	Tip        lipgloss.Style
	HeadStatus lipgloss.Style
}

// NewBranchStyles initializes and returns all styles used in the branch command
func NewBranchStyles() BranchStyles {
	// Base container style
	containerStyle := AppStyle.
		PaddingLeft(2).
		PaddingRight(2)

	return BranchStyles{
		Container: containerStyle,

		// Headers use the base header style
		Header: HeaderStyle.
			Background(TerminalMagenta).
			Padding(0, 1),
		SubHeader: HeaderStyle.
			Foreground(TerminalWhite).
			Background(TerminalBlue).
			Padding(0, 1),
		ListHeader: BaseTextStyle.
			Foreground(TerminalMagenta).
			MarginBottom(1).
			Padding(0, 1),

		// Branch and VM styles
		BranchName: BranchNameStyle,
		VMID:       VmIDStyle,
		CurrentState: SecondaryTextStyle.
			Foreground(TerminalWhite),

		// Status styles
		Progress: MutedTextStyle.
			Italic(true).
			Padding(1, 0),
		Success: PrimaryTextStyle.
			Foreground(TerminalGreen).
			Bold(true).
			Padding(1, 0),
		Warning: PrimaryTextStyle.
			Foreground(TerminalYellow).
			Bold(true).
			Padding(1, 0),
		Error: ErrorTextStyle.
			Padding(1, 0),

		// Information styles
		Info: PrimaryTextStyle.
			Foreground(TerminalWhite).
			Padding(0, 1),
		InfoLabel: SecondaryTextStyle.
			Foreground(TerminalSilver).
			Width(12),
		InfoValue: PrimaryTextStyle.
			Foreground(TerminalWhite),
		ListItem: PrimaryTextStyle.
			PaddingLeft(3),
		Tip: MutedTextStyle.
			Italic(true).
			Foreground(TerminalGray),
		HeadStatus: HeadStatusStyle,
	}
}
