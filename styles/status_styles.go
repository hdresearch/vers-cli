package styles

import (
	"github.com/charmbracelet/lipgloss"
)

// StatusStyles contains all styles used in the status command
type StatusStyles struct {
	Container       lipgloss.Style
	HeadStatus      lipgloss.Style
	ClusterInfo     lipgloss.Style
	VMListHeader    lipgloss.Style
	ClusterName     lipgloss.Style
	ClusterListItem lipgloss.Style
	ClusterData     lipgloss.Style
	VMInfo          lipgloss.Style
	NoData          lipgloss.Style
	Tip             lipgloss.Style
	VMID            lipgloss.Style
}

// NewStatusStyles initializes and returns all styles used in the status command
func NewStatusStyles() StatusStyles {
	containerStyle := AppStyle

	listItemStyle := containerStyle.
		Inherit(SecondaryTextStyle).
		Padding(0, 0)
	dataItemStyle := PrimaryTextStyle.
		Foreground(TerminalWhite)

	return StatusStyles{
		Container:  containerStyle,
		HeadStatus: HeadStatusStyle,
		ClusterInfo: containerStyle.
			Inherit(PrimaryTextStyle).
			Padding(0, 1),
		VMListHeader: containerStyle.
			Inherit(PrimaryTextStyle).
			PaddingBottom(1),
		ClusterName: listItemStyle.
			Inherit(HeaderStyle).
			Background(TerminalBlue).
			Foreground(TerminalWhite).
			Padding(0, 1),
		ClusterListItem: listItemStyle.
			MarginBottom(1).
			BorderStyle(lipgloss.RoundedBorder()).
			BorderForeground(BorderColor).
			BorderRight(true).
			BorderBottom(true),
		ClusterData: dataItemStyle.
			PaddingLeft(2).
			PaddingRight(1),
		VMInfo: listItemStyle,
		NoData: MutedTextStyle.
			Padding(1, 0),
		Tip:  HelpStyle.Padding(0, 0),
		VMID: VmIDStyle,
	}
}
