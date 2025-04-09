package styles

import "github.com/charmbracelet/lipgloss"

// StyleForState returns a style based on some condition (e.g., focused, blurred)
func StyleForState(baseStyle lipgloss.Style, isSelected bool) lipgloss.Style {
	if isSelected {
		// Example: Invert or apply different background/foreground
		return baseStyle.Foreground(Background).Background(Foreground)
	}
	return baseStyle
}