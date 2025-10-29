package tui

import (
	"strings"

	tea "github.com/charmbracelet/bubbletea"
)

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case initLoadMsg:
		m.loading = true
		return m, loadClustersCmd(m)

	case clustersLoadedMsg:
		m.loading = false
		if msg.err != nil {
			m.status = "Failed to load clusters: " + msg.err.Error()
			return m, nil
		}
		m.clusterBacking = msg.raw
		m.clusters.SetItems(msg.items)
		if len(msg.items) > 0 {
			m.clusters.Select(0)
			cid := m.clusterBacking[0].ID
			m.loading = true
			return m, loadVMsCmd(m, cid)
		}
		return m, nil

	case vmsLoadedMsg:
		m.loading = false
		if msg.err != nil {
			m.status = "Failed to load VMs: " + msg.err.Error()
			return m, nil
		}
		m.vms.SetItems(msg.items)
		if len(msg.items) > 0 {
			m.vms.Select(0)
		}
		return m, nil

	case actionCompletedMsg:
		if msg.err != nil {
			m.status = msg.text + ": " + msg.err.Error()
		} else {
			m.status = msg.text
		}
		m.loading = true
		return m, refreshCurrentVMsCmd(m)
	case historyLoadedMsg:
		m.status = ""
		if msg.err != nil {
			m.status = "History error: " + msg.err.Error()
			m.modalActive = false
			return m, nil
		}
		m.modalText = msg.lines
		m.modalActive = true
		m.modalKind = "text"
		m.modalPrompt = "History:"
		return m, nil
	case treeLoadedMsg:
		m.status = ""
		if msg.err != nil {
			m.status = "Tree error: " + msg.err.Error()
			m.modalActive = false
			return m, nil
		}
		m.modalText = msg.lines
		m.modalActive = true
		m.modalKind = "text"
		m.modalPrompt = "Tree:"
		return m, nil

	case tea.KeyMsg:
		switch msg.String() {
		case "q", "ctrl+c":
			return m, tea.Quit
		case "tab":
			if m.focus == focusClusters {
				m.setFocus(focusVMs)
			} else {
				m.setFocus(focusClusters)
			}
			return m, nil
		}
		// pass keys to active list
		if m.focus == focusClusters {
			var cmd tea.Cmd
			m.clusters, cmd = m.clusters.Update(msg)
			idx := m.clusters.Index()
			if idx != m.prevClusterIdx && idx >= 0 && idx < len(m.clusterBacking) {
				m.prevClusterIdx = idx
				m.loading = true
				cid := m.clusterBacking[idx].ID
				return m, loadVMsCmd(m, cid)
			}
			// Note: Cluster actions removed - cluster concept has been removed from API
			// VM list now shows all VMs directly
			return m, cmd
		}
		if m.focus == focusVMs {
			// action keys on VMs
			switch msg.String() {
			case "enter":
				if it, ok := m.vms.SelectedItem().(vmItem); ok {
					m.status = "Connecting..."
					return m, doConnectCmd(m, it.ID)
				}
			case "b":
				if it, ok := m.vms.SelectedItem().(vmItem); ok {
					m.modalActive = true
					m.modalKind = "input"
					m.modalPrompt = "Branch alias:"
					m.input.SetValue("")
					id := it.ID
					m.onSubmit = func(alias string) tea.Cmd {
						m.modalActive = false
						m.status = "Branching..."
						return doBranchCmd(m, id, alias)
					}
					return m, nil
				}
			case "c":
				if it, ok := m.vms.SelectedItem().(vmItem); ok {
					m.modalActive = true
					m.modalKind = "input"
					m.modalPrompt = "Commit tags (comma-separated):"
					m.input.Placeholder = "tag1, tag2"
					m.input.SetValue("")
					id := it.ID
					m.onSubmit = func(tagCSV string) tea.Cmd {
						m.modalActive = false
						m.status = "Committing..."
						return doCommitCmd(m, id, tagCSV)
					}
					return m, nil
				}
			case "p":
				if it, ok := m.vms.SelectedItem().(vmItem); ok {
					m.modalActive = true
					m.modalKind = "confirm"
					m.modalPrompt = "Pause VM '" + it.TitleText + "'?"
					id := it.ID
					m.onConfirm = func() tea.Cmd { m.modalActive = false; m.status = "Pausing..."; return doPauseCmd(m, id) }
					return m, nil
				}
			case "r":
				if it, ok := m.vms.SelectedItem().(vmItem); ok {
					m.modalActive = true
					m.modalKind = "confirm"
					m.modalPrompt = "Resume VM '" + it.TitleText + "'?"
					id := it.ID
					m.onConfirm = func() tea.Cmd { m.modalActive = false; m.status = "Resuming..."; return doResumeCmd(m, id) }
					return m, nil
				}
			case "k":
				if it, ok := m.vms.SelectedItem().(vmItem); ok {
					m.modalActive = true
					m.modalKind = "confirm"
					m.recursiveVMKill = false
					m.modalPrompt = "Delete VM '" + it.TitleText + "'? (press 'R' to toggle recursive: off)"
					id := it.ID
					m.onConfirm = func() tea.Cmd {
						m.modalActive = false
						m.status = "Deleting..."
						return doKillVMCmd(m, id, m.recursiveVMKill)
					}
					return m, nil
				}
			case "h":
				if it, ok := m.vms.SelectedItem().(vmItem); ok {
					m.modalActive = true
					m.modalKind = "text"
					m.modalPrompt = "History:"
					m.status = "Loading history..."
					return m, loadHistoryCmd(m, it.ID)
				}
			}
			var cmd tea.Cmd
			m.vms, cmd = m.vms.Update(msg)
			return m, cmd
		}
	case tea.WindowSizeMsg:
		m.width, m.height = msg.Width, msg.Height
		// give lists some space
		m.clusters.SetSize(msg.Width/2-2, msg.Height-6)
		m.vms.SetSize(msg.Width/2-2, msg.Height-6)
		return m, nil
	}
	// modal input/confirm handling
	if m.modalActive {
		if m.modalKind == "input" {
			if km, ok := msg.(tea.KeyMsg); ok {
				if km.Type == tea.KeyEnter {
					if m.onSubmit != nil {
						return m, m.onSubmit(m.input.Value())
					}
				}
				if km.Type == tea.KeyEsc {
					m.modalActive = false
					return m, nil
				}
			}
			var cmd tea.Cmd
			m.input, cmd = m.input.Update(msg)
			return m, cmd
		}
		if m.modalKind == "confirm" {
			if km, ok := msg.(tea.KeyMsg); ok {
				if (km.String() == "R" || km.String() == "r") && m.modalPrompt != "" {
					// toggle recursive for VM delete prompt
					m.recursiveVMKill = !m.recursiveVMKill
					onoff := "off"
					if m.recursiveVMKill {
						onoff = "on"
					}
					// preserve VM text before bracket
					parts := strings.SplitN(m.modalPrompt, "?", 2)
					base := m.modalPrompt
					if len(parts) > 0 {
						base = parts[0] + "?"
					}
					m.modalPrompt = base + " (press 'R' to toggle recursive: " + onoff + ")"
					return m, nil
				}
				if km.String() == "y" || km.Type == tea.KeyEnter {
					if m.onConfirm != nil {
						return m, m.onConfirm()
					}
				}
				if km.String() == "n" || km.Type == tea.KeyEsc {
					m.modalActive = false
					return m, nil
				}
			}
			return m, nil
		}
		if m.modalKind == "text" {
			if km, ok := msg.(tea.KeyMsg); ok {
				if km.Type == tea.KeyEsc || km.String() == "q" {
					m.modalActive = false
					return m, nil
				}
			}
			return m, nil
		}
	}

	// spinner tick
	var cmd tea.Cmd
	m.spin, cmd = m.spin.Update(msg)
	return m, cmd
}
