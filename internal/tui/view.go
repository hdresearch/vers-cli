package tui

import (
	"github.com/charmbracelet/lipgloss"
	"strings"
)

func (m Model) View() string {
	var b strings.Builder
	// header
	b.WriteString("Vers TUI\n\n")
	// single VM list with focused style
	focused := lipgloss.NewStyle().Padding(0, 1).Border(lipgloss.RoundedBorder()).BorderForeground(lipgloss.Color("63"))
	vmBox := focused
	vmView := vmBox.Render(m.vms.View())
	b.WriteString(vmView)
	b.WriteString("\n")
	// modal overlay
	if m.modalActive {
		b.WriteString("\n")
		if m.modalKind == "confirm" {
			b.WriteString("[y]es / [n]o: " + m.modalPrompt + "\n")
		} else if m.modalKind == "input" {
			b.WriteString(m.modalPrompt + "\n")
			b.WriteString(m.input.View() + "\n")
			b.WriteString("Enter to submit • Esc to cancel\n")
		} else if m.modalKind == "text" {
			b.WriteString(m.modalPrompt + "\n")
			for _, ln := range m.modalText {
				b.WriteString(ln + "\n")
			}
			b.WriteString("Esc/q to close\n")
		}
	}
	// status/footer with help
	b.WriteString("\n")
	if m.loading {
		b.WriteString(m.spin.View() + " Loading...\n")
	}
	if m.status != "" {
		b.WriteString(m.status + "\n")
	}
	b.WriteString(m.help.View(m.keys))
	b.WriteString("\n")
	return b.String()
}
