package tui

import (
    "github.com/hdresearch/vers-cli/internal/app"
    tea "github.com/charmbracelet/bubbletea"
)

func Run(a *app.App) error {
    p := tea.NewProgram(New(a), tea.WithAltScreen())
    _, err := p.Run()
    return err
}

