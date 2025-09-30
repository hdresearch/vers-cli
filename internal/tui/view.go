package tui

import (
	"github.com/charmbracelet/lipgloss"
	"strings"
)

func (m Model) View() string {
	var b strings.Builder
	// header
	b.WriteString("Vers TUI\n\n")
	// columns with simple lipgloss columns
	// focused styles cached in model
	focused := m.boxFocused
	blurred := m.boxBlurred
	leftBox := blurred
	rightBox := blurred
	if m.focus == focusClusters {
		leftBox = focused
	}
	if m.focus == focusVMs {
		rightBox = focused
	}

	if m.showClusters {
		left := leftBox.Render(m.clusters.View())
		right := rightBox.Render(m.vms.View())
		row := lipgloss.JoinHorizontal(lipgloss.Top, left, right)
		b.WriteString(row)
	} else {
		// Sidebar hidden; always focus VMs visually
		b.WriteString(focused.Render(m.vms.View()))
	}
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
	if !m.showClusters {
		// Small hint to show the sidebar again
		b.WriteString("s: show sidebar\n")
	}
	if m.showHelp {
		b.WriteString(m.help.View(m.keys))
		b.WriteString("\n")
	}
	return b.String()
}
