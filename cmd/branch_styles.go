package cmd

import (
	"github.com/charmbracelet/lipgloss"
	"github.com/hdresearch/vers-cli/styles"
)

// BranchStyles contains all styles used in the branch command
type BranchStyles struct {
	// Base container style
	Container lipgloss.Style

	// Headers and titles
	Header      lipgloss.Style
	SubHeader   lipgloss.Style
	ListHeader  lipgloss.Style

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
	Info      lipgloss.Style
	InfoLabel lipgloss.Style
	InfoValue lipgloss.Style
	ListItem  lipgloss.Style
	Tip       lipgloss.Style
	HeadStatus lipgloss.Style
}

// NewBranchStyles initializes and returns all styles used in the branch command
func NewBranchStyles() BranchStyles {
	// Base container style
	containerStyle := styles.AppStyle.
		PaddingLeft(2).
		PaddingRight(2)

	return BranchStyles{
		Container: containerStyle,

		// Headers use the base header style
		Header: styles.HeaderStyle.
			Background(styles.TerminalMagenta).
			Padding(0, 1),
		SubHeader: styles.HeaderStyle.
			Foreground(styles.TerminalWhite).
			Background(styles.TerminalBlue).
			Padding(0, 1),
		ListHeader: styles.BaseTextStyle.
			Foreground(styles.TerminalMagenta).
			MarginBottom(1).
			Padding(0, 1),

		// Branch and VM styles
		BranchName: styles.BranchNameStyle,
		VMID: styles.VmIDStyle,
		CurrentState: styles.SecondaryTextStyle.
			Foreground(styles.TerminalWhite),

		// Status styles
		Progress: styles.MutedTextStyle.
			Italic(true).
			Padding(1, 0),
		Success: styles.PrimaryTextStyle.
			Foreground(styles.TerminalGreen).
			Bold(true).
			Padding(1, 0),
		Warning: styles.PrimaryTextStyle.
			Foreground(styles.TerminalYellow).
			Bold(true).
			Padding(1, 0),
		Error: styles.ErrorTextStyle.
			Padding(1, 0),

		// Information styles
		Info: styles.PrimaryTextStyle.
			Foreground(styles.TerminalWhite).
			Padding(0, 1),
		InfoLabel: styles.SecondaryTextStyle.
			Foreground(styles.TerminalSilver).
			Width(12),
		InfoValue: styles.PrimaryTextStyle.
			Foreground(styles.TerminalWhite),
		ListItem: styles.PrimaryTextStyle.
			PaddingLeft(3),
		Tip: styles.MutedTextStyle.
			Italic(true).
			Foreground(styles.TerminalGray),
		HeadStatus: styles.HeadStatusStyle,
	}
} 