package tui

import (
	"context"
	tea "github.com/charmbracelet/bubbletea"
	delsvc "github.com/hdresearch/vers-cli/internal/services/deletion"
	"strings"
)

// applyLayout sets list sizes based on current width/height, focus, and sidebar visibility.
func applyLayout(m Model) Model {
	listH := m.height - 6
	if listH < 5 {
		listH = m.height
	}
	if !m.showClusters {
		// Hide clusters; give most width to VMs
		m.clusters.SetSize(0, listH)
		m.vms.SetSize(max(10, m.width-2), listH)
		return m
	}
	if m.focus == focusVMs {
		// Narrow sidebar when VM pane focused
		cw := clamp(m.width/4, 22, 30)
		m.clusters.SetSize(cw, listH)
		m.vms.SetSize(max(10, m.width-cw-2), listH)
		return m
	}
	// Focus on clusters: split roughly 50/50
	m.clusters.SetSize(m.width/2-2, listH)
	m.vms.SetSize(m.width/2-2, listH)
	return m
}

func clamp(v, lo, hi int) int {
	if v < lo {
		return lo
	}
	if v > hi {
		return hi
	}
	return v
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width, m.height = msg.Width, msg.Height
		m = applyLayout(m)
		return m, nil

	case tea.KeyMsg:
		// If a modal is active, it must capture key input before any
		// list navigation or action handlers (e.g., Enter triggering SSH).
		if m.modalActive {
			// input modal: forward keys to textinput and handle submit/cancel
			if m.modalKind == "input" {
				if msg.Type == tea.KeyEnter {
					if m.onSubmit != nil {
						// blur input when submitting
						m.input.Blur()
						return m, m.onSubmit(m.input.Value())
					}
				}
				if msg.Type == tea.KeyEsc {
					m.modalActive = false
					m.input.Blur()
					return m, nil
				}
				var cmd tea.Cmd
				m.input, cmd = m.input.Update(msg)
				return m, cmd
			}
			// confirm modal: y/enter confirms, n/esc cancels; 'R' toggles recursive delete state
			if m.modalKind == "confirm" {
				if (msg.String() == "R" || msg.String() == "r") && m.modalPrompt != "" {
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
				if msg.String() == "y" || msg.Type == tea.KeyEnter {
					if m.onConfirm != nil {
						return m, m.onConfirm()
					}
				}
				if msg.String() == "n" || msg.Type == tea.KeyEsc {
					m.modalActive = false
					return m, nil
				}
				return m, nil
			}
			// text modal: esc/q closes
			if m.modalKind == "text" {
				if msg.Type == tea.KeyEsc || msg.String() == "q" {
					m.modalActive = false
					return m, nil
				}
				return m, nil
			}
		}

		// No modal active: handle global keys and list actions
		// If sidebar is hidden but focus is still on clusters, migrate focus to VMs.
		if !m.showClusters && m.focus == focusClusters {
			m.setFocus(focusVMs)
		}
		// Global keybindings (no modal active)
		switch msg.String() {
		case "q", "ctrl+c":
			return m, tea.Quit
		case "tab":
			if m.focus == focusClusters {
				m.setFocus(focusVMs)
			} else {
				m.setFocus(focusClusters)
			}
			m = applyLayout(m)
			return m, nil
		case "s":
			// toggle sidebar visibility; ensure VM focus if hiding
			m.showClusters = !m.showClusters
			if !m.showClusters && m.focus == focusClusters {
				m.setFocus(focusVMs)
			}
			m = applyLayout(m)
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
			// cluster actions
			switch msg.String() {
			case "k":
				if cid, ok := m.selectedClusterID(); ok {
					m.modalActive = true
					m.modalKind = "confirm"
					m.modalPrompt = "Delete cluster '" + cid + "'?"
					id := cid
					m.onConfirm = func() tea.Cmd {
						m.modalActive = false
						m.status = "Deleting cluster..."
						return func() tea.Msg {
							ctx, cancel := context.WithTimeout(context.Background(), m.app.Timeouts.APIMedium)
							defer cancel()
							_, err := delsvc.DeleteCluster(ctx, m.app.Client, id)
							if err != nil {
								return actionCompletedMsg{text: "Cluster delete failed", err: err}
							}
							return actionCompletedMsg{text: "Cluster deleted", err: nil}
						}
					}
					return m, nil
				}
			case "t":
				if cid, ok := m.selectedClusterID(); ok {
					m.status = "Loading tree..."
					return m, loadTreeCmd(m, cid)
				}
			}
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
					m.input.Placeholder = "alias"
					m.input.CursorStart()
					m.input.Focus()
					id := it.ID
					m.onSubmit = func(alias string) tea.Cmd {
						m.modalActive = false
						m.input.Blur()
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
					m.input.CursorStart()
					m.input.Focus()
					id := it.ID
					m.onSubmit = func(tagCSV string) tea.Cmd {
						m.modalActive = false
						m.input.Blur()
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
		// nothing handled; continue to spinner update
		return m, nil

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

		// nothing handled; continue to spinner update
		return m, nil
	}

	// spinner tick
	var cmd tea.Cmd
	m.spin, cmd = m.spin.Update(msg)
	return m, cmd
}
