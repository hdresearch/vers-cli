package tui

import (
    "context"
    "fmt"
    "strings"

    tea "github.com/charmbracelet/bubbletea"
    "github.com/charmbracelet/bubbles/list"
    "github.com/charmbracelet/bubbles/spinner"
    "github.com/charmbracelet/bubbles/textinput"
    "github.com/charmbracelet/bubbles/help"
    "github.com/hdresearch/vers-cli/internal/app"
    "github.com/hdresearch/vers-cli/internal/handlers"
    svcstatus "github.com/hdresearch/vers-cli/internal/services/status"
    histSvc "github.com/hdresearch/vers-cli/internal/services/history"
    treeSvc "github.com/hdresearch/vers-cli/internal/services/tree"
    delsvc "github.com/hdresearch/vers-cli/internal/services/deletion"
    vmSvc "github.com/hdresearch/vers-cli/internal/services/vm"
    sshutil "github.com/hdresearch/vers-cli/internal/ssh"
    "github.com/hdresearch/vers-cli/internal/utils"
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
func (i clusterItem) FilterValue() string { if i.Alias != "" { return i.Alias }; return i.ID }

type vmItem struct{ TitleText, DescText, ID, Alias, State string }
func (i vmItem) Title() string       { return i.TitleText }
func (i vmItem) Description() string { return i.DescText }
func (i vmItem) FilterValue() string { if i.Alias != "" { return i.Alias }; return i.ID }

// messages
type initLoadMsg struct{}
type clustersLoadedMsg struct{ items []list.Item; raw []svcCluster; err error }
type vmsLoadedMsg struct{ clusterID string; items []list.Item; err error }
type actionCompletedMsg struct{ text string; err error }
type historyLoadedMsg struct{ lines []string; err error }
type treeLoadedMsg struct{ lines []string; err error }

// raw backing info we may need
type svcCluster struct{ ID, Alias string }

type Model struct {
    app *app.App

    focus    focus
    clusters list.Model
    vms      list.Model

    clusterBacking []svcCluster
    prevClusterIdx int

    loading        bool
    spin           spinner.Model
    status         string

    width  int
    height int

    // modal state
    modalActive bool
    modalKind   string // confirm | input
    modalPrompt string
    onConfirm   func() tea.Cmd
    input       textinput.Model
    onSubmit    func(string) tea.Cmd

    help  help.Model
    keys  keyMap
    modalText []string

    recursiveVMKill bool
}

func New(appc *app.App) Model {
    lclusters := list.New(nil, list.NewDefaultDelegate(), 40, 12)
    lclusters.Title = "Clusters"
    lvms := list.New(nil, list.NewDefaultDelegate(), 60, 12)
    lvms.Title = "VMs"
    sp := spinner.New(); sp.Spinner = spinner.Line
    ti := textinput.New(); ti.Placeholder = "alias"; ti.CharLimit = 64
    m := Model{app: appc, focus: focusClusters, clusters: lclusters, vms: lvms, spin: sp, input: ti, help: help.New(), keys: defaultKeys()}
    m.setFocus(focusClusters)
    return m
}

func (m *Model) setFocus(f focus) {
    m.focus = f
    m.clusters.SetFilteringEnabled(true)
    m.vms.SetFilteringEnabled(true)
}

func (m Model) Init() tea.Cmd { return tea.Batch(func() tea.Msg { return initLoadMsg{} }, m.spin.Tick) }

// commands
func loadClustersCmd(m Model) tea.Cmd {
    return func() tea.Msg {
        ctx, cancel := context.WithTimeout(context.Background(), m.app.Timeouts.APIMedium)
        defer cancel()
        rows, err := svcstatus.ListClusters(ctx, m.app.Client)
        if err != nil { return clustersLoadedMsg{err: err} }
        items := make([]list.Item, 0, len(rows))
        backing := make([]svcCluster, 0, len(rows))
        for _, c := range rows {
            disp := c.Alias; if disp == "" { disp = c.ID }
            root := c.RootVmID
            // best-effort root alias lookup
            for _, v := range c.Vms { if v.ID == c.RootVmID && v.Alias != "" { root = v.Alias; break } }
            items = append(items, clusterItem{TitleText: disp, DescText: fmt.Sprintf("Root: %s | VMs: %d", root, c.VmCount), ID: c.ID, Alias: c.Alias})
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
        if err != nil { return vmsLoadedMsg{clusterID: clusterID, err: err} }
        items := make([]list.Item, 0, len(cl.Vms))
        for _, v := range cl.Vms {
            disp := v.Alias; if disp == "" { disp = v.ID }
            items = append(items, vmItem{TitleText: disp, DescText: fmt.Sprintf("State: %s", v.State), ID: v.ID, Alias: v.Alias, State: string(v.State)})
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
        if err != nil { return actionCompletedMsg{text: "Pause failed", err: err} }
        return actionCompletedMsg{text: "Paused", err: nil}
    }
}

func doResumeCmd(m Model, vmID string) tea.Cmd {
    return func() tea.Msg {
        ctx, cancel := context.WithTimeout(context.Background(), m.app.Timeouts.APIMedium)
        defer cancel()
        _, err := handlers.HandleResume(ctx, m.app, handlers.ResumeReq{Target: vmID})
        if err != nil { return actionCompletedMsg{text: "Resume failed", err: err} }
        return actionCompletedMsg{text: "Resumed", err: nil}
    }
}

func doBranchCmd(m Model, vmID, alias string) tea.Cmd {
    return func() tea.Msg {
        ctx, cancel := context.WithTimeout(context.Background(), m.app.Timeouts.APIMedium)
        defer cancel()
        _, err := handlers.HandleBranch(ctx, m.app, handlers.BranchReq{Target: vmID, Alias: alias, Checkout: false})
        if err != nil { return actionCompletedMsg{text: "Branch failed", err: err} }
        return actionCompletedMsg{text: "Branched", err: nil}
    }
}

func doConnectCmd(m Model, vmID string) tea.Cmd {
    return func() tea.Msg {
        // Resolve SSH connection info first (fast API call)
        ctx, cancel := context.WithTimeout(context.Background(), m.app.Timeouts.APIMedium)
        defer cancel()
        info, err := vmSvc.GetConnectInfo(ctx, m.app.Client, vmID)
        if err != nil { return actionCompletedMsg{text: "SSH failed", err: err} }

        // Determine host/port (DNAT vs local route)
        sshHost := info.Host
        sshPort := fmt.Sprintf("%d", info.VM.NetworkInfo.SSHPort)
        if utils.IsHostLocal(info.Host) {
            sshHost = info.VM.IPAddress
            sshPort = "22"
        }

        // Use Bubble Tea ExecProcess to release the terminal during SSH
        cmd := sshutil.SSHCommand(sshHost, sshPort, info.KeyPath)
        return tea.ExecProcess(cmd, func(err error) tea.Msg {
            if err != nil { return actionCompletedMsg{text: "SSH failed", err: err} }
            return actionCompletedMsg{text: "SSH session ended", err: nil}
        })()
    }
}

func refreshCurrentVMsCmd(m Model) tea.Cmd {
    idx := m.clusters.Index()
    if idx < 0 || idx >= len(m.clusterBacking) { return nil }
    cid := m.clusterBacking[idx].ID
    return loadVMsCmd(m, cid)
}

// helpers
func (m Model) selectedVMID() (string, bool) {
    if it, ok := m.vms.SelectedItem().(vmItem); ok { return it.ID, true }
    return "", false
}

func (m Model) selectedClusterID() (string, bool) {
    idx := m.clusters.Index()
    if idx >= 0 && idx < len(m.clusterBacking) { return m.clusterBacking[idx].ID, true }
    return "", false
}

func loadHistoryCmd(m Model, vmID string) tea.Cmd {
    return func() tea.Msg {
        ctx, cancel := context.WithTimeout(context.Background(), m.app.Timeouts.APIMedium)
        defer cancel()
        commits, err := histSvc.GetCommits(ctx, m.app.Client, vmID)
        if err != nil { return historyLoadedMsg{err: err} }
        lines := make([]string, 0, len(commits))
        for _, c := range commits {
            line := c.ID
            if c.Alias != "" { line = c.Alias + " (" + c.ID + ")" }
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
        if err != nil { return treeLoadedMsg{err: err} }
        // simple tree render
        vmMap := map[string]vers.VmDto{}
        for _, v := range cl.Vms { vmMap[v.ID] = v }
        var lines []string
        var walk func(id string, prefix string, isLast bool)
        walk = func(id string, prefix string, isLast bool) {
            v := vmMap[id]
            name := v.Alias; if name == "" { name = v.ID }
            connector := "├── "; if isLast { connector = "└── " }
            lines = append(lines, prefix+connector+name+" ["+string(v.State)+"]")
            childPrefix := prefix; if isLast { childPrefix += "    " } else { childPrefix += "│   " }
            for i, cid := range v.Children { walk(cid, childPrefix, i == len(v.Children)-1) }
        }
        if cl.RootVmID != "" { walk(cl.RootVmID, "", true) }
        return treeLoadedMsg{lines: lines}
    }
}

func doCommitCmd(m Model, vmID string, tagCSV string) tea.Cmd {
    return func() tea.Msg {
        // Split and trim tags
        tags := []string{}
        for _, t := range strings.Split(tagCSV, ",") {
            tt := strings.TrimSpace(t)
            if tt != "" { tags = append(tags, tt) }
        }
        ctx, cancel := context.WithTimeout(context.Background(), m.app.Timeouts.APILong)
        defer cancel()
        // direct SDK call mirrors cmd/commit.go
        body := vers.APIVmCommitParams{ VmCommitRequest: vers.VmCommitRequestParam{ Tags: vers.F(tags) } }
        // If we cannot resolve alias to id, the API accepts id only; the TUI uses vmID
        _, err := m.app.Client.API.Vm.Commit(ctx, vmID, body)
        if err != nil { return actionCompletedMsg{text: "Commit failed", err: err} }
        return actionCompletedMsg{text: "Committed", err: nil}
    }
}

func doKillVMCmd(m Model, vmID string, recursive bool) tea.Cmd {
    return func() tea.Msg {
        ctx, cancel := context.WithTimeout(context.Background(), m.app.Timeouts.APIMedium)
        defer cancel()
        _, err := delsvc.DeleteVM(ctx, m.app.Client, vmID, recursive)
        if err != nil { return actionCompletedMsg{text: "Delete failed", err: err} }
        return actionCompletedMsg{text: "Deleted", err: nil}
    }
}
