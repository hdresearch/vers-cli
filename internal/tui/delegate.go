package tui

import (
	"io"

	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/hdresearch/vers-cli/styles"
)

// vmDelegate renders VMs on a single line with a colored state badge.
type vmDelegate struct{}

func newVMDelegate() list.ItemDelegate { return vmDelegate{} }

func (d vmDelegate) Height() int                               { return 1 }
func (d vmDelegate) Spacing() int                              { return 0 }
func (d vmDelegate) Update(msg tea.Msg, m *list.Model) tea.Cmd { return nil }

func (d vmDelegate) Render(w io.Writer, m list.Model, index int, listItem list.Item) {
	it, ok := listItem.(vmItem)
	if !ok {
		return
	}
	// Build single-line text: tree title + dimmed separator + colored state
	title := it.TitleText
	state := it.State
	sep := lipgloss.NewStyle().Foreground(styles.Muted).Render(" • ")
	stateStyled := styleState(state).Render(state)

	// Style title based on selection
	var titleStyled string
	if index == m.Index() {
		titleStyled = lipgloss.NewStyle().Bold(true).Render(title)
	} else {
		titleStyled = lipgloss.NewStyle().Foreground(styles.Muted).Render(title)
	}

	_, _ = io.WriteString(w, titleStyled)
	_, _ = io.WriteString(w, sep)
	_, _ = io.WriteString(w, stateStyled)
}

func styleState(state string) lipgloss.Style {
	switch state {
	case "Running":
		return lipgloss.NewStyle().Foreground(styles.Primary)
	case "Paused":
		return lipgloss.NewStyle().Foreground(styles.Muted)
	case "Stopped", "Deleted", "Error":
		return lipgloss.NewStyle().Foreground(styles.Error)
	default:
		return lipgloss.NewStyle().Foreground(styles.Muted)
	}
}
