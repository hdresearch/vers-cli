package tui

import (
    "strings"
    "github.com/charmbracelet/lipgloss"
)

func (m Model) View() string {
    var b strings.Builder
    // header
    b.WriteString("Vers TUI\n\n")
    // columns with simple lipgloss columns
    // focused styles
    focused := lipgloss.NewStyle().Padding(0, 1).Border(lipgloss.RoundedBorder()).BorderForeground(lipgloss.Color("63"))
    blurred := lipgloss.NewStyle().Padding(0, 1).Border(lipgloss.RoundedBorder()).BorderForeground(lipgloss.Color("240"))
    leftBox := blurred
    rightBox := blurred
    if m.focus == focusClusters { leftBox = focused }
    if m.focus == focusVMs { rightBox = focused }

    left := leftBox.Render(m.clusters.View())
    right := rightBox.Render(m.vms.View())
    row := lipgloss.JoinHorizontal(lipgloss.Top, left, right)
    b.WriteString(row)
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
            for _, ln := range m.modalText { b.WriteString(ln + "\n") }
            b.WriteString("Esc/q to close\n")
        }
    }
    // status/footer with help
    b.WriteString("\n")
    if m.loading { b.WriteString(m.spin.View() + " Loading...\n") }
    if m.status != "" { b.WriteString(m.status + "\n") }
    b.WriteString(m.help.View(m.keys))
    b.WriteString("\n")
    return b.String()
}
