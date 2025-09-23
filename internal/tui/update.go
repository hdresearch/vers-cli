package tui

import (
	"context"
	tea "github.com/charmbracelet/bubbletea"
	delsvc "github.com/hdresearch/vers-cli/internal/services/deletion"
	"strings"
	"time"
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
		// keep background refresh ticking
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
				// debounce VM reload to avoid hammering API while scrolling
				m.clusterReqSeq++
				seq := m.clusterReqSeq
				return m, tea.Batch(
					tea.Tick(120*time.Millisecond, func(time.Time) tea.Msg { return vmReloadDebouncedMsg{clusterID: cid, seq: seq} }),
					m.spin.Tick,
				)
			}
			// cluster actions
			switch msg.String() {
			case "a":
				// Rename cluster: prompt for new alias, prefill with current alias
				if idx >= 0 && idx < len(m.clusterBacking) {
					cur := m.clusterBacking[idx]
					m.modalActive = true
					m.modalKind = "input"
					m.modalPrompt = "New cluster alias:"
					m.input.Placeholder = "alias"
					m.input.SetValue(cur.Alias)
					m.input.CursorEnd()
					m.input.Focus()
					id := cur.ID
					m.onSubmit = func(alias string) tea.Cmd {
						m.input.Blur()
						// If alias is unchanged or empty, just close
						if alias == cur.Alias || alias == "" {
							m.modalActive = false
							return nil
						}
						// Duplicate check among clusters
						var conflict string
						for _, c := range m.clusterBacking {
							if c.ID != id && c.Alias == alias && alias != "" {
								conflict = c.ID
								break
							}
						}
						if conflict != "" {
							// Ask for confirmation before overwriting
							m.modalKind = "confirm"
							m.modalPrompt = "Alias '" + alias + "' is used by cluster '" + conflict + "'. Overwrite?"
							m.onConfirm = func() tea.Cmd {
								m.modalActive = false
								m.status = "Renaming cluster..."
								return tea.Batch(doRenameClusterCmd(m, id, alias), loadClustersCmd(m))
							}
							return nil
						}
						// No conflict: proceed
						m.modalActive = false
						m.status = "Renaming cluster..."
						return tea.Batch(doRenameClusterCmd(m, id, alias), loadClustersCmd(m))
					}
					return m, nil
				}
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
			case "a":
				if it, ok := m.vms.SelectedItem().(vmItem); ok {
					m.modalActive = true
					m.modalKind = "input"
					m.modalPrompt = "New VM alias:"
					m.input.Placeholder = "alias"
					// Prefill with current alias (if empty, leave blank)
					m.input.SetValue(it.Alias)
					m.input.CursorEnd()
					m.input.Focus()
					id := it.ID
					m.onSubmit = func(alias string) tea.Cmd {
						m.input.Blur()
						// If alias unchanged or empty, just close
						if alias == it.Alias || alias == "" {
							m.modalActive = false
							return nil
						}
						// Duplicate check within visible VM list
						var conflict string
						items := m.vms.Items()
						for i := 0; i < len(items); i++ {
							if vi, ok := items[i].(vmItem); ok {
								if vi.ID != id && vi.Alias == alias && alias != "" {
									if vi.Alias != "" {
										conflict = vi.Alias
									} else {
										conflict = vi.ID
									}
									break
								}
							}
						}
						if conflict != "" {
							// Confirm overwrite before proceeding
							m.modalKind = "confirm"
							m.modalPrompt = "Alias '" + alias + "' is used by VM '" + conflict + "'. Overwrite?"
							m.onConfirm = func() tea.Cmd {
								m.modalActive = false
								m.status = "Renaming..."
								return doRenameVMCmd(m, id, alias)
							}
							return nil
						}
						// No conflict
						m.modalActive = false
						m.status = "Renaming..."
						return doRenameVMCmd(m, id, alias)
					}
					return m, nil
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
		// nothing handled; continue
		return m, nil

	case initLoadMsg:
		m.loading = true
		return m, tea.Batch(loadClustersCmd(m), m.spin.Tick, scheduleRefreshCmd(3*time.Second))

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
		// diff against fingerprint; only update items if changed
		// Compute fingerprint
		fpb := strings.Builder{}
		for _, it := range msg.items {
			if v, ok := it.(vmItem); ok {
				fpb.WriteString(v.ID)
				fpb.WriteByte('|')
				fpb.WriteString(v.Alias)
				fpb.WriteByte('|')
				fpb.WriteString(v.State)
				fpb.WriteByte(';')
			}
		}
		fp := fpb.String()
		if fp != m.lastVmsFingerprint {
			// preserve selection by ID if possible
			selID, _ := m.selectedVMID()
			m.vms.SetItems(msg.items)
			if selID != "" {
				for i, it := range m.vms.Items() {
					if vi, ok := it.(vmItem); ok && vi.ID == selID {
						m.vms.Select(i)
						break
					}
				}
			} else if len(msg.items) > 0 {
				m.vms.Select(0)
			}
			m.lastVmsFingerprint = fp
		}
		m.loading = false
		if msg.err != nil {
			m.status = "Failed to load VMs: " + msg.err.Error()
			return m, nil
		}
		return m, nil

	case actionCompletedMsg:
		if msg.err != nil {
			m.status = msg.text + ": " + msg.err.Error()
		} else {
			m.status = msg.text
		}
		m.loading = true
		return m, tea.Batch(refreshCurrentVMsCmd(m), m.spin.Tick)
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

	case vmReloadDebouncedMsg:
		// Only proceed if the sequence matches the latest request and the
		// selected cluster hasn't changed since the timer was set.
		if msg.seq == m.clusterReqSeq {
			// Verify currently selected cluster still matches
			idx := m.clusters.Index()
			if idx >= 0 && idx < len(m.clusterBacking) && m.clusterBacking[idx].ID == msg.clusterID {
				return m, loadVMsCmd(m, msg.clusterID)
			}
		}
		return m, nil

	case refreshTickMsg:
		// schedule next tick
		cmdNext := scheduleRefreshCmd(3 * time.Second)
		// Quiet refresh (no spinner) if we have a selected cluster and no modal
		if m.modalActive {
			return m, cmdNext
		}
		idx := m.clusters.Index()
		if idx >= 0 && idx < len(m.clusterBacking) {
			cid := m.clusterBacking[idx].ID
			return m, tea.Batch(loadVMsCmd(m, cid), cmdNext)
		}
		return m, cmdNext

		// nothing handled; continue to spinner update
		return m, nil
	}

	// spinner tick only when loading to reduce idle work
	if m.loading {
		var cmd tea.Cmd
		m.spin, cmd = m.spin.Update(msg)
		return m, cmd
	}
	return m, nil
}
