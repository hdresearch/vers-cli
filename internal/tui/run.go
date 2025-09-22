package tui

import (
	tea "github.com/charmbracelet/bubbletea"
	"github.com/hdresearch/vers-cli/internal/app"
)

func Run(a *app.App) error {
	p := tea.NewProgram(New(a), tea.WithAltScreen())
	_, err := p.Run()
	return err
}
