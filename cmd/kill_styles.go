package cmd

import (
	"github.com/charmbracelet/lipgloss"
	"github.com/hdresearch/vers-cli/styles"
)

// KillStyles contains all styles used in the kill command
type KillStyles struct {
	Container      lipgloss.Style
	HeadStatus     lipgloss.Style
	Error          lipgloss.Style
	Warning        lipgloss.Style
	Progress       lipgloss.Style
	Success        lipgloss.Style
	NoData         lipgloss.Style
}

// NewKillStyles initializes and returns all styles used in the kill command
func NewKillStyles() KillStyles {
	containerStyle := styles.AppStyle

	return KillStyles{
		Container:  containerStyle,
		HeadStatus: styles.HeadStatusStyle,
		Error:      styles.ErrorTextStyle,
		Warning:    styles.ErrorTextStyle.Foreground(styles.TerminalYellow),
		Progress:   styles.PrimaryTextStyle,
		Success:    styles.PrimaryTextStyle.
		Padding(1,0).
		Foreground(styles.TerminalGreen),
		NoData:     styles.MutedTextStyle.Padding(1, 0),
	}
} 