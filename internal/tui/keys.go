package tui

import (
	"github.com/charmbracelet/bubbles/key"
)

type keyMap struct {
	Quit    key.Binding
	Switch  key.Binding
	Left    key.Binding
	Right   key.Binding
	Connect key.Binding
	Branch  key.Binding
	Rename  key.Binding
	Pause   key.Binding
	Resume  key.Binding
	Kill    key.Binding
	Commit  key.Binding
	History key.Binding
	Sidebar key.Binding
}

func defaultKeys() keyMap {
	return keyMap{
		Quit:    key.NewBinding(key.WithKeys("q", "ctrl+c"), key.WithHelp("q", "quit")),
		Switch:  key.NewBinding(key.WithKeys("tab"), key.WithHelp("tab", "switch panel")),
		Left:    key.NewBinding(key.WithKeys("h"), key.WithHelp("h", "focus left")),
		Right:   key.NewBinding(key.WithKeys("l"), key.WithHelp("l", "focus right")),
		Connect: key.NewBinding(key.WithKeys("enter"), key.WithHelp("enter", "connect / load")),
		Branch:  key.NewBinding(key.WithKeys("b"), key.WithHelp("b", "branch VM")),
		Rename:  key.NewBinding(key.WithKeys("a"), key.WithHelp("a", "rename alias")),
		Pause:   key.NewBinding(key.WithKeys("p"), key.WithHelp("p", "pause VM")),
		Resume:  key.NewBinding(key.WithKeys("r"), key.WithHelp("r", "resume VM")),
		Kill:    key.NewBinding(key.WithKeys("k"), key.WithHelp("k", "delete")),
		Commit:  key.NewBinding(key.WithKeys("c"), key.WithHelp("c", "commit VM")),
		History: key.NewBinding(key.WithKeys("h"), key.WithHelp("h", "history")),
		Sidebar: key.NewBinding(key.WithKeys("s"), key.WithHelp("s", "toggle sidebar")),
	}
}

// Implement help.KeyMap interface for bubbles/help
func (k keyMap) ShortHelp() []key.Binding {
	return []key.Binding{k.Connect, k.Branch, k.Commit, k.Rename, k.Pause, k.Resume, k.History, k.Left, k.Right, k.Sidebar, k.Switch, k.Quit}
}

func (k keyMap) FullHelp() [][]key.Binding {
	return [][]key.Binding{
		{k.Connect, k.Branch, k.Rename, k.Commit},
		{k.Pause, k.Resume, k.Kill},
		{k.History, k.Left, k.Right, k.Sidebar},
		{k.Switch, k.Quit},
	}
}
