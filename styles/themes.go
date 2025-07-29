package styles

import "github.com/charmbracelet/lipgloss"

var (
	// Base Styles
	AppStyle = lipgloss.NewStyle().Padding(1, 2) // Base padding for the whole app view

	BaseTextStyle = lipgloss.NewStyle().Foreground(Foreground)

	// Semantic Text Styles (inherit from BaseTextStyle)
	PrimaryTextStyle   = BaseTextStyle.Foreground(Primary)
	SecondaryTextStyle = BaseTextStyle.Foreground(Secondary)
	MutedTextStyle     = BaseTextStyle.Foreground(Muted)
	ErrorTextStyle     = BaseTextStyle.Bold(true).Foreground(Error)

	// Component Styles
	HeaderStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(PrimaryFg).
			Background(Primary).
			Padding(0, 1)

	StatusStyle = lipgloss.NewStyle().
			Inherit(AppStyle). // Get base padding
			BorderStyle(lipgloss.RoundedBorder()).
			BorderForeground(BorderColor).
			Padding(0, 1)

	SelectedListItemStyle = lipgloss.NewStyle().
				Foreground(PrimaryFg).
				Background(PrimaryDim). // Use a dimmed primary for selection
				Padding(0, 1)

	NormalListItemStyle = lipgloss.NewStyle().
				Padding(0, 1) // Basic padding for alignment

	HelpStyle = MutedTextStyle.Padding(1, 0)

	// Version Control Styles
	HeadStatusStyle = BaseTextStyle.
			Foreground(Primary).
			Italic(true).
			Padding(1, 0)

	BranchNameStyle = BaseTextStyle.
			Bold(true).
			Foreground(Primary)

	VmIDStyle = BaseTextStyle.
			Foreground(TerminalLime)
)
