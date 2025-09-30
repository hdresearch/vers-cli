package tui

import (
	"context"
	"strconv"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/help"
	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/hdresearch/vers-cli/internal/app"
	"github.com/hdresearch/vers-cli/internal/handlers"
	delsvc "github.com/hdresearch/vers-cli/internal/services/deletion"
	histSvc "github.com/hdresearch/vers-cli/internal/services/history"
	svcstatus "github.com/hdresearch/vers-cli/internal/services/status"
	treeSvc "github.com/hdresearch/vers-cli/internal/services/tree"
	vmSvc "github.com/hdresearch/vers-cli/internal/services/vm"
	sshutil "github.com/hdresearch/vers-cli/internal/ssh"
	"github.com/hdresearch/vers-cli/internal/utils"
	"github.com/hdresearch/vers-cli/styles"
	vers "github.com/hdresearch/vers-sdk-go"
)

type focus int

const (
	focusClusters focus = iota
	focusVMs
	focusModal
)

// cluster and vm list items
type clusterItem struct{ TitleText, DescText, ID, Alias string }

func (i clusterItem) Title() string       { return i.TitleText }
func (i clusterItem) Description() string { return i.DescText }
func (i clusterItem) FilterValue() string {
	if i.Alias != "" {
		return i.Alias
	}
	return i.ID
}

type vmItem struct {
	TitleText, DescText, ID, Alias, State string
	Depth                                 int
}

func (i vmItem) Title() string       { return i.TitleText }
func (i vmItem) Description() string { return i.DescText }
func (i vmItem) FilterValue() string {
	if i.Alias != "" {
		return i.Alias
	}
	return i.ID
}

// messages
type initLoadMsg struct{}
type clustersLoadedMsg struct {
	items []list.Item
	raw   []svcCluster
	err   error
}
type vmsLoadedMsg struct {
	clusterID string
	items     []list.Item
	err       error
}
type vmReloadDebouncedMsg struct {
	clusterID string
	seq       int
}
type actionCompletedMsg struct {
	text string
	err  error
}
type historyLoadedMsg struct {
	lines []string
	err   error
}
type treeLoadedMsg struct {
	lines []string
	err   error
}
type refreshTickMsg struct{}

// raw backing info we may need
type svcCluster struct{ ID, Alias string }

// branchState holds context for the two-step branch create flow
type branchState struct {
	vmID     string
	alias    string
	checkout bool
}

type Model struct {
	app *app.App

	focus    focus
	clusters list.Model
	vms      list.Model

	// layout
	showClusters bool

	clusterBacking []svcCluster
	prevClusterIdx int

	loading bool
	spin    spinner.Model
	status  string

	width  int
	height int
	// cached styles
	boxFocused lipgloss.Style
	boxBlurred lipgloss.Style

	// modal state
	modalActive bool
	modalKind   string // confirm | input
	modalPrompt string
	onConfirm   func() tea.Cmd
	input       textinput.Model
	onSubmit    func(string) tea.Cmd

	help      help.Model
	keys      keyMap
	modalText []string

	recursiveVMKill bool

	// branching
	branch branchState

	// help visibility
	showHelp bool

	// debounce state for VM loading when cluster selection changes
	clusterReqSeq int

	// background refresh and diffing
	lastVmsFingerprint string
}

func New(appc *app.App) Model {
	lclusters := list.New(nil, list.NewDefaultDelegate(), 40, 12)
	lclusters.Title = "Clusters"
	lvms := list.New(nil, newVMDelegate(), 60, 12)
	lvms.Title = "VMs"
	sp := spinner.New()
	sp.Spinner = spinner.Line
	ti := textinput.New()
	ti.Placeholder = "alias"
	ti.CharLimit = 64
	m := Model{app: appc, focus: focusClusters, clusters: lclusters, vms: lvms, spin: sp, input: ti, help: help.New(), keys: defaultKeys(), showClusters: true, showHelp: true}
	m.help.ShowAll = false
	// cache box styles to avoid allocating every render
	m.boxFocused = lipgloss.NewStyle().Padding(0, 1).Border(lipgloss.RoundedBorder()).BorderForeground(styles.Primary)
	m.boxBlurred = lipgloss.NewStyle().Padding(0, 1).Border(lipgloss.RoundedBorder()).BorderForeground(styles.BorderColor)
	m.setFocus(focusClusters)
	return m
}

func (m *Model) setFocus(f focus) {
	m.focus = f
	m.clusters.SetFilteringEnabled(true)
	m.vms.SetFilteringEnabled(true)
}

func (m Model) Init() tea.Cmd { return tea.Batch(func() tea.Msg { return initLoadMsg{} }, m.spin.Tick) }

// scheduleRefreshCmd returns a Tick that fires a refresh message after d.
func scheduleRefreshCmd(d time.Duration) tea.Cmd {
	return tea.Tick(d, func(time.Time) tea.Msg { return refreshTickMsg{} })
}

// commands
func loadClustersCmd(m Model) tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), m.app.Timeouts.APIMedium)
		defer cancel()
		rows, err := svcstatus.ListClusters(ctx, m.app.Client)
		if err != nil {
			return clustersLoadedMsg{err: err}
		}
		items := make([]list.Item, 0, len(rows))
		backing := make([]svcCluster, 0, len(rows))
		for _, c := range rows {
			disp := c.Alias
			if disp == "" {
				disp = c.ID
			}
			root := c.RootVmID
			// best-effort root alias lookup
			for _, v := range c.Vms {
				if v.ID == c.RootVmID && v.Alias != "" {
					root = v.Alias
					break
				}
			}
			// Build description without fmt.Sprintf
			var db strings.Builder
			db.Grow(len(root) + 16)
			db.WriteString("Root: ")
			db.WriteString(root)
			db.WriteString(" | VMs: ")
			db.WriteString(strconv.FormatInt(c.VmCount, 10))
			items = append(items, clusterItem{TitleText: disp, DescText: db.String(), ID: c.ID, Alias: c.Alias})
			backing = append(backing, svcCluster{ID: c.ID, Alias: c.Alias})
		}
		return clustersLoadedMsg{items: items, raw: backing}
	}
}

func loadVMsCmd(m Model, clusterID string) tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), m.app.Timeouts.APIMedium)
		defer cancel()
		cl, err := svcstatus.GetCluster(ctx, m.app.Client, clusterID)
		if err != nil {
			return vmsLoadedMsg{clusterID: clusterID, err: err}
		}
		// Build lineage-ordered list similar to the tree view
		vmMap := map[string]vers.VmDto{}
		for _, v := range cl.Vms {
			vmMap[v.ID] = v
		}
		var items []list.Item
		var walk func(id string, prefix string, isLast bool, depth int)
		walk = func(id string, prefix string, isLast bool, depth int) {
			v := vmMap[id]
			name := v.Alias
			if name == "" {
				name = v.ID
			}
			connector := "├── "
			if isLast {
				connector = "└── "
			}
			title := name
			if depth > 0 {
				title = prefix + connector + name
			}
			items = append(items, vmItem{TitleText: title, DescText: "", ID: v.ID, Alias: v.Alias, State: string(v.State), Depth: depth})
			childPrefix := prefix
			if depth > 0 {
				if isLast {
					childPrefix += "    "
				} else {
					childPrefix += "│   "
				}
			}
			for i, cid := range v.Children {
				walk(cid, childPrefix, i == len(v.Children)-1, depth+1)
			}
		}
		if cl.RootVmID != "" {
			walk(cl.RootVmID, "", true, 0)
		}
		return vmsLoadedMsg{clusterID: cl.ID, items: items}
	}
}

// action commands
func doPauseCmd(m Model, vmID string) tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), m.app.Timeouts.APIMedium)
		defer cancel()
		_, err := handlers.HandlePause(ctx, m.app, handlers.PauseReq{Target: vmID})
		if err != nil {
			return actionCompletedMsg{text: "Pause failed", err: err}
		}
		return actionCompletedMsg{text: "Paused", err: nil}
	}
}

func doResumeCmd(m Model, vmID string) tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), m.app.Timeouts.APIMedium)
		defer cancel()
		_, err := handlers.HandleResume(ctx, m.app, handlers.ResumeReq{Target: vmID})
		if err != nil {
			return actionCompletedMsg{text: "Resume failed", err: err}
		}
		return actionCompletedMsg{text: "Resumed", err: nil}
	}
}

func doBranchCmd(m Model, vmID, alias string) tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), m.app.Timeouts.APIMedium)
		defer cancel()
		_, err := handlers.HandleBranch(ctx, m.app, handlers.BranchReq{Target: vmID, Alias: alias, Checkout: false})
		if err != nil {
			return actionCompletedMsg{text: "Branch failed", err: err}
		}
		return actionCompletedMsg{text: "Branched", err: nil}
	}
}

func doRenameVMCmd(m Model, vmID, alias string) tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), m.app.Timeouts.APIMedium)
		defer cancel()
		_, err := handlers.HandleRename(ctx, m.app, handlers.RenameReq{IsCluster: false, Target: vmID, NewAlias: alias})
		if err != nil {
			return actionCompletedMsg{text: "Rename failed", err: err}
		}
		return actionCompletedMsg{text: "Renamed", err: nil}
	}
}

func doRenameClusterCmd(m Model, clusterID, alias string) tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), m.app.Timeouts.APIMedium)
		defer cancel()
		_, err := handlers.HandleRename(ctx, m.app, handlers.RenameReq{IsCluster: true, Target: clusterID, NewAlias: alias})
		if err != nil {
			return actionCompletedMsg{text: "Cluster rename failed", err: err}
		}
		return actionCompletedMsg{text: "Cluster renamed", err: nil}
	}
}

func doConnectCmd(m Model, vmID string) tea.Cmd {
	return func() tea.Msg {
		// Resolve SSH connection info first (fast API call)
		ctx, cancel := context.WithTimeout(context.Background(), m.app.Timeouts.APIMedium)
		defer cancel()
		info, err := vmSvc.GetConnectInfo(ctx, m.app.Client, vmID)
		if err != nil {
			return actionCompletedMsg{text: "SSH failed", err: err}
		}

		// Determine host/port (DNAT vs local route)
		sshHost := info.Host
		sshPort := strconv.FormatInt(info.VM.NetworkInfo.SSHPort, 10)
		if utils.IsHostLocal(info.Host) {
			sshHost = info.VM.IPAddress
			sshPort = "22"
		}

		// Use Bubble Tea ExecProcess to release the terminal during SSH
		cmd := sshutil.SSHCommand(sshHost, sshPort, info.KeyPath)
		return tea.ExecProcess(cmd, func(err error) tea.Msg {
			if err != nil {
				return actionCompletedMsg{text: "SSH failed", err: err}
			}
			return actionCompletedMsg{text: "SSH session ended", err: nil}
		})()
	}
}

func refreshCurrentVMsCmd(m Model) tea.Cmd {
	idx := m.clusters.Index()
	if idx < 0 || idx >= len(m.clusterBacking) {
		return nil
	}
	cid := m.clusterBacking[idx].ID
	return loadVMsCmd(m, cid)
}

// helpers
func (m Model) selectedVMID() (string, bool) {
	if it, ok := m.vms.SelectedItem().(vmItem); ok {
		return it.ID, true
	}
	return "", false
}

func (m Model) selectedClusterID() (string, bool) {
	idx := m.clusters.Index()
	if idx >= 0 && idx < len(m.clusterBacking) {
		return m.clusterBacking[idx].ID, true
	}
	return "", false
}

func loadHistoryCmd(m Model, vmID string) tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), m.app.Timeouts.APIMedium)
		defer cancel()
		commits, err := histSvc.GetCommits(ctx, m.app.Client, vmID)
		if err != nil {
			return historyLoadedMsg{err: err}
		}
		lines := make([]string, 0, len(commits))
		for _, c := range commits {
			line := c.ID
			if c.Alias != "" {
				line = c.Alias + " (" + c.ID + ")"
			}
			line += " | " + c.Author
			lines = append(lines, line)
		}
		return historyLoadedMsg{lines: lines}
	}
}

func loadTreeCmd(m Model, clusterID string) tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), m.app.Timeouts.APIMedium)
		defer cancel()
		cl, err := treeSvc.GetClusterByIdentifier(ctx, m.app.Client, clusterID)
		if err != nil {
			return treeLoadedMsg{err: err}
		}
		// simple tree render
		vmMap := map[string]vers.VmDto{}
		for _, v := range cl.Vms {
			vmMap[v.ID] = v
		}
		var lines []string
		var walk func(id string, prefix string, isLast bool)
		walk = func(id string, prefix string, isLast bool) {
			v := vmMap[id]
			name := v.Alias
			if name == "" {
				name = v.ID
			}
			connector := "├── "
			if isLast {
				connector = "└── "
			}
			lines = append(lines, prefix+connector+name+" ["+string(v.State)+"]")
			childPrefix := prefix
			if isLast {
				childPrefix += "    "
			} else {
				childPrefix += "│   "
			}
			for i, cid := range v.Children {
				walk(cid, childPrefix, i == len(v.Children)-1)
			}
		}
		if cl.RootVmID != "" {
			walk(cl.RootVmID, "", true)
		}
		return treeLoadedMsg{lines: lines}
	}
}

func doCommitCmd(m Model, vmID string, tagCSV string) tea.Cmd {
	return func() tea.Msg {
		// Split and trim tags
		tags := []string{}
		for _, t := range strings.Split(tagCSV, ",") {
			tt := strings.TrimSpace(t)
			if tt != "" {
				tags = append(tags, tt)
			}
		}
		ctx, cancel := context.WithTimeout(context.Background(), m.app.Timeouts.APILong)
		defer cancel()
		// direct SDK call mirrors cmd/commit.go
		body := vers.APIVmCommitParams{VmCommitRequest: vers.VmCommitRequestParam{Tags: vers.F(tags)}}
		// If we cannot resolve alias to id, the API accepts id only; the TUI uses vmID
		_, err := m.app.Client.API.Vm.Commit(ctx, vmID, body)
		if err != nil {
			return actionCompletedMsg{text: "Commit failed", err: err}
		}
		return actionCompletedMsg{text: "Committed", err: nil}
	}
}

func doKillVMCmd(m Model, vmID string, recursive bool) tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), m.app.Timeouts.APIMedium)
		defer cancel()
		_, err := delsvc.DeleteVM(ctx, m.app.Client, vmID, recursive)
		if err != nil {
			return actionCompletedMsg{text: "Delete failed", err: err}
		}
		return actionCompletedMsg{text: "Deleted", err: nil}
	}
}
