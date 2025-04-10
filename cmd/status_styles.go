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
	VMListHeader   lipgloss.Style
	ClusterName    lipgloss.Style
	ClusterListItem lipgloss.Style
	ClusterData    lipgloss.Style
	VMInfo         lipgloss.Style
	NoData         lipgloss.Style
	Tip            lipgloss.Style
	VMID           lipgloss.Style
}

// NewStatusStyles initializes and returns all styles used in the status command
func NewStatusStyles() StatusStyles {
	containerStyle := styles.AppStyle

	listItemStyle := containerStyle.
		Inherit(styles.SecondaryTextStyle).
		Padding(0,0)
	dataItemStyle := styles.PrimaryTextStyle.
		Foreground(styles.TerminalWhite)

	return StatusStyles{
		Container: containerStyle,
		ClusterHeader: styles.HeaderStyle,
		ClusterInfo: containerStyle.
			Inherit(styles.PrimaryTextStyle).
			Padding(0, 1),
		VMListHeader: containerStyle.
			Inherit(styles.PrimaryTextStyle).
			PaddingBottom(1),
		ClusterName: listItemStyle.
			Inherit(styles.HeaderStyle).	
			Background(styles.TerminalBlue).
			Foreground(styles.TerminalWhite).
			Padding(0,1),	
		ClusterListItem: listItemStyle.
			MarginBottom(1).
			BorderStyle(lipgloss.RoundedBorder()).
			BorderForeground(styles.BorderColor).
			BorderRight(true).
			BorderBottom(true),
		ClusterData: dataItemStyle.
			PaddingLeft(2),
		VMInfo: listItemStyle,
		NoData: styles.MutedTextStyle.
			Padding(1,0),
		Tip: styles.HelpStyle.Padding(0,0),
		VMID: dataItemStyle.
			Foreground(styles.TerminalYellow),
	}
} 