package tui

import (
	"github.com/charmbracelet/bubbles/key"
)

type keyMap struct {
	Quit    key.Binding
	Switch  key.Binding
	Connect key.Binding
	Branch  key.Binding
	Pause   key.Binding
	Resume  key.Binding
	Kill    key.Binding
	Commit  key.Binding
	Tree    key.Binding
}

func defaultKeys() keyMap {
	return keyMap{
		Quit:    key.NewBinding(key.WithKeys("q", "ctrl+c"), key.WithHelp("q", "quit")),
		Switch:  key.NewBinding(key.WithKeys("tab"), key.WithHelp("tab", "switch panel")),
		Connect: key.NewBinding(key.WithKeys("enter"), key.WithHelp("enter", "connect / load")),
		Branch:  key.NewBinding(key.WithKeys("b"), key.WithHelp("b", "branch VM")),
		Pause:   key.NewBinding(key.WithKeys("p"), key.WithHelp("p", "pause VM")),
		Resume:  key.NewBinding(key.WithKeys("r"), key.WithHelp("r", "resume VM")),
		Kill:    key.NewBinding(key.WithKeys("k"), key.WithHelp("k", "delete")),
		Commit:  key.NewBinding(key.WithKeys("c"), key.WithHelp("c", "commit VM")),
		Tree:    key.NewBinding(key.WithKeys("t"), key.WithHelp("t", "tree")),
	}
}

// Implement help.KeyMap interface for bubbles/help
func (k keyMap) ShortHelp() []key.Binding {
	return []key.Binding{k.Connect, k.Branch, k.Pause, k.Resume, k.Tree, k.Switch, k.Quit}
}

func (k keyMap) FullHelp() [][]key.Binding {
	return [][]key.Binding{
		{k.Connect, k.Branch, k.Commit},
		{k.Pause, k.Resume, k.Kill},
		{k.Tree, k.Switch, k.Quit},
	}
}
