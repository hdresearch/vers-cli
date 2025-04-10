package cmd

import (
	"github.com/charmbracelet/lipgloss"
	"github.com/hdresearch/vers-cli/styles"
)

// StatusStyles contains all styles used in the status command
type StatusStyles struct {
	Container      lipgloss.Style
	ClusterHeader  lipgloss.Style
	ClusterInfo    lipgloss.Style
	ClusterList    lipgloss.Style
	VMListHeader   lipgloss.Style
	VMInfo         lipgloss.Style
	NoData         lipgloss.Style
	Tip            lipgloss.Style
}

// NewStatusStyles initializes and returns all styles used in the status command
func NewStatusStyles() StatusStyles {
	containerStyle := styles.AppStyle

	listItemStyle := containerStyle.
		Inherit(styles.SecondaryTextStyle).
		Padding(0, 1)

	return StatusStyles{
		Container: containerStyle,
		ClusterHeader: styles.HeaderStyle,
		ClusterInfo: containerStyle.
			Inherit(styles.PrimaryTextStyle).
			Padding(0, 1),
		ClusterList: listItemStyle,
		VMListHeader: containerStyle.
			Inherit(styles.PrimaryTextStyle).
			Padding(1, 0),
		VMInfo: listItemStyle,
		NoData: containerStyle.
			Inherit(styles.MutedTextStyle),
		Tip: containerStyle.
			Inherit(styles.HelpStyle),
	}
} 