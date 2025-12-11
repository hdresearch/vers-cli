package tui

import (
	"context"
	"fmt"
	"io"
	"strings"

	"github.com/charmbracelet/bubbles/help"
	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/hdresearch/vers-cli/internal/app"
	"github.com/hdresearch/vers-cli/internal/handlers"
	delsvc "github.com/hdresearch/vers-cli/internal/services/deletion"
	histSvc "github.com/hdresearch/vers-cli/internal/services/history"
	svcstatus "github.com/hdresearch/vers-cli/internal/services/status"
	vmSvc "github.com/hdresearch/vers-cli/internal/services/vm"
	sshutil "github.com/hdresearch/vers-cli/internal/ssh"
)

type focus int

const (
	focusVMs focus = iota
	focusModal
)

// VM list item
type vmItem struct{ TitleText, DescText, ID, Alias string }

func (i vmItem) Title() string       { return i.TitleText }
func (i vmItem) Description() string { return i.DescText }
func (i vmItem) FilterValue() string {
	return i.ID
}

// messages
type initLoadMsg struct{}
type vmsLoadedMsg struct {
	items []list.Item
	err   error
}
type actionCompletedMsg struct {
	text string
	err  error
}
type historyLoadedMsg struct {
	lines []string
	err   error
}

type Model struct {
	app *app.App

	focus focus
	vms   list.Model

	loading bool
	spin    spinner.Model
	status  string

	width  int
	height int

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
}

func New(appc *app.App) Model {
	lvms := list.New(nil, list.NewDefaultDelegate(), 60, 12)
	lvms.Title = "VMs"
	sp := spinner.New()
	sp.Spinner = spinner.Line
	ti := textinput.New()
	ti.Placeholder = "alias"
	ti.CharLimit = 64
	m := Model{app: appc, focus: focusVMs, vms: lvms, spin: sp, input: ti, help: help.New(), keys: defaultKeys()}
	m.setFocus(focusVMs)
	return m
}

func (m *Model) setFocus(f focus) {
	m.focus = f
	m.vms.SetFilteringEnabled(true)
}

func (m Model) Init() tea.Cmd { return tea.Batch(func() tea.Msg { return initLoadMsg{} }, m.spin.Tick) }

// commands
func loadVMsCmd(m Model) tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), m.app.Timeouts.APIMedium)
		defer cancel()
		rows, err := svcstatus.ListVMs(ctx, m.app.Client)
		if err != nil {
			return vmsLoadedMsg{err: err}
		}
		items := make([]list.Item, 0, len(rows))
		for _, vm := range rows {
			disp := vm.VmID
			items = append(items, vmItem{TitleText: disp, DescText: fmt.Sprintf("Parent: %s", vm.Parent), ID: vm.VmID, Alias: ""})
		}
		return vmsLoadedMsg{items: items}
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

func doConnectCmd(m Model, vmID string) tea.Cmd {
	return func() tea.Msg {
		// Resolve SSH connection info first (fast API call)
		ctx, cancel := context.WithTimeout(context.Background(), m.app.Timeouts.APIMedium)
		defer cancel()
		info, err := vmSvc.GetConnectInfo(ctx, m.app.Client, vmID)
		if err != nil {
			return actionCompletedMsg{text: "SSH failed", err: err}
		}

		// Use VM ID as host for SSH-over-TLS
		sshHost := info.Host

		// Use native SSH client via Bubble Tea Exec
		// We wrap the SSH interactive session in an exec.Cmd-like interface
		return tea.Exec(&sshExecCmd{
			client: sshutil.NewClient(sshHost, info.KeyPath),
		}, func(err error) tea.Msg {
			if err != nil {
				return actionCompletedMsg{text: "SSH failed", err: err}
			}
			return actionCompletedMsg{text: "SSH session ended", err: nil}
		})()
	}
}

// sshExecCmd implements tea.ExecCommand for native SSH sessions.
type sshExecCmd struct {
	client *sshutil.Client
	stdin  io.Reader
	stdout io.Writer
	stderr io.Writer
}

func (c *sshExecCmd) Run() error {
	return c.client.Interactive(context.Background(), c.stdin, c.stdout, c.stderr)
}

func (c *sshExecCmd) SetStdin(r io.Reader)  { c.stdin = r }
func (c *sshExecCmd) SetStdout(w io.Writer) { c.stdout = w }
func (c *sshExecCmd) SetStderr(w io.Writer) { c.stderr = w }

func refreshVMsCmd(m Model) tea.Cmd {
	return loadVMsCmd(m)
}

// helpers
func (m Model) selectedVMID() (string, bool) {
	if it, ok := m.vms.SelectedItem().(vmItem); ok {
		return it.ID, true
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
		// direct SDK call - note: tags no longer supported in new SDK
		_, err := m.app.Client.Vm.Commit(ctx, vmID)
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
