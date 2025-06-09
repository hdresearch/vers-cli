package styles

import (
	"github.com/charmbracelet/lipgloss"
)

// KillStyles contains all styles used in the kill command
type KillStyles struct {
	Container  lipgloss.Style
	HeadStatus lipgloss.Style
	Error      lipgloss.Style
	Warning    lipgloss.Style
	Progress   lipgloss.Style
	Success    lipgloss.Style
	NoData     lipgloss.Style
}

// NewKillStyles initializes and returns all styles used in the kill command
func NewKillStyles() KillStyles {
	containerStyle := AppStyle

	return KillStyles{
		Container:  containerStyle,
		HeadStatus: HeadStatusStyle,
		Error:      ErrorTextStyle,
		Warning:    ErrorTextStyle.Foreground(TerminalYellow),
		Progress:   PrimaryTextStyle,
		Success: PrimaryTextStyle.
			Padding(1, 0).
			Foreground(TerminalGreen),
		NoData: MutedTextStyle.Padding(1, 0),
	}
}
