package app

import (
	"fmt"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/mwdomino/cascade/internal/action"
	"github.com/mwdomino/cascade/internal/config"
	"github.com/mwdomino/cascade/internal/editor"
	"github.com/mwdomino/cascade/internal/model"
	"github.com/mwdomino/cascade/internal/store"
	"github.com/mwdomino/cascade/internal/theme"
	"github.com/mwdomino/cascade/internal/tui/breadcrumb"
	"github.com/mwdomino/cascade/internal/tui/details"
	"github.com/mwdomino/cascade/internal/tui/keys"
	"github.com/mwdomino/cascade/internal/tui/palette"
	"github.com/mwdomino/cascade/internal/tui/prompt"
	"github.com/mwdomino/cascade/internal/tui/search"
	"github.com/mwdomino/cascade/internal/tui/sidebar"
	"github.com/sahilm/fuzzy"
)

type promptMode int

const (
	promptNone        promptMode = iota
	promptNew
	promptQuickNew
	promptRename
	promptMoveTo
	promptSearchLocal
	promptSearchGlobal
)

type editorClosedMsg struct{}

type Model struct {
	Tree    *store.Tree
	Theme   *theme.Theme
	Cfg     *config.Config
	Keys    keys.Map
	Current *model.Node // parent of the displayed siblings
	Cursor  int
	ShowDone bool

	Sidebar    sidebar.Model
	Details    details.Model
	Breadcrumb breadcrumb.Model

	PromptMode promptMode
	Prompt     prompt.Model

	// Search state
	LocalQuery    string
	GlobalMatches []*model.Node
	GlobalMode    bool

	// Confirm overlay state
	ConfirmMode bool
	ConfirmHard bool
	PendingDD   bool
	DDDeadline  time.Time

	// Palette + action state
	ActionReg   *action.Registry
	PaletteMode bool
	Palette     palette.Model
	ActionOut   *action.Result
	ActionByKey map[string]action.Action

	Width, Height int
}

func New(tree *store.Tree, th *theme.Theme, cfg *config.Config, reg *action.Registry) tea.Model {
	byKey := make(map[string]action.Action)
	for name, def := range reg.Defs() {
		if def.Keybind != "" {
			byKey[def.Keybind] = action.Action{Name: name, Def: def}
		}
	}
	return &Model{
		Tree:        tree,
		Theme:       th,
		Cfg:         cfg,
		Keys:        keys.Default(),
		Current:     tree.Root,
		Sidebar:     sidebar.Model{Theme: th},
		Details:     details.Model{Theme: th},
		Breadcrumb:  breadcrumb.Model{Theme: th},
		Prompt:      prompt.New(th),
		ActionReg:   reg,
		ActionByKey: byKey,
	}
}

func (m *Model) Init() tea.Cmd { return nil }

func (m *Model) visibleSiblings() []*model.Node {
	if m.GlobalMode {
		return m.GlobalMatches
	}
	sibs := m.Current.Children
	if m.LocalQuery != "" {
		sibs = search.LocalFilter(sibs, m.LocalQuery)
	}
	if !m.ShowDone {
		filtered := make([]*model.Node, 0, len(sibs))
		for _, n := range sibs {
			if n.FM.Status != model.StatusDone {
				filtered = append(filtered, n)
			}
		}
		sibs = filtered
	}
	return sibs
}

func (m *Model) selectedNode() *model.Node {
	sibs := m.visibleSiblings()
	if len(sibs) == 0 || m.Cursor < 0 || m.Cursor >= len(sibs) {
		return nil
	}
	return sibs[m.Cursor]
}

// restoreNav attempts to restore Current and Cursor to the saved paths after a
// tree reload. Falls back to root/0 if the paths no longer exist.
func (m *Model) restoreNav(savedCurrent, savedSelected string) {
	restored := m.Tree.NodeAt(savedCurrent)
	if restored == nil {
		m.Current = m.Tree.Root
		m.Cursor = 0
		return
	}
	m.Current = restored
	if savedSelected != "" {
		sel := m.Tree.NodeAt(savedSelected)
		if sel != nil {
			for i, c := range m.Current.Children {
				if c == sel {
					m.Cursor = i
					return
				}
			}
		}
	}
	m.Cursor = 0
}

func (m *Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case editorClosedMsg:
		savedCurrent := m.Current.Path
		savedSelected := ""
		if sel := m.selectedNode(); sel != nil {
			savedSelected = sel.Path
		}
		_ = m.Tree.Reload()
		m.restoreNav(savedCurrent, savedSelected)
		return m, nil
	case tea.WindowSizeMsg:
		m.Width = msg.Width
		m.Height = msg.Height
		sw := msg.Width / 5
		if sw < 20 {
			sw = 20
		}
		m.Sidebar.Width = sw
		m.Sidebar.Height = msg.Height - 2
		m.Details.Width = msg.Width - sw - 2
		m.Details.Height = msg.Height - 2
	case tea.KeyMsg:
		// Clear transient action output on any keypress.
		if m.ActionOut != nil {
			m.ActionOut = nil
			return m, nil
		}

		// Palette mode: forward keys to palette, handle Esc.
		if m.PaletteMode {
			if msg.String() == "esc" {
				m.PaletteMode = false
				return m, nil
			}
			var cmd tea.Cmd
			m.Palette, cmd = m.Palette.Update(msg)
			return m, cmd
		}

		// Confirm overlay: handle before everything else.
		if m.ConfirmMode {
			switch msg.String() {
			case "y", "Y", "enter":
				n := m.selectedNode()
				if n != nil {
					var err error
					if m.ConfirmHard {
						err = m.Tree.HardDelete(n)
					} else {
						err = m.Tree.SoftDelete(n)
					}
					if err == nil && m.Cursor >= len(m.Current.Children) && m.Cursor > 0 {
						m.Cursor--
					}
				}
				m.ConfirmMode = false
			case "n", "N", "esc":
				m.ConfirmMode = false
			}
			return m, nil
		}

		// When prompt is active, forward all keys to the prompt except Enter/Esc.
		if m.PromptMode != promptNone {
			switch msg.String() {
			case "enter":
				return m, m.submitPrompt()
			case "esc":
				m.PromptMode = promptNone
				m.Prompt.Blur()
				return m, nil
			}
			var cmd tea.Cmd
			m.Prompt, cmd = m.Prompt.Update(msg)
			return m, cmd
		}

		// Esc outside prompt clears active search state.
		if msg.String() == "esc" && (m.LocalQuery != "" || m.GlobalMode) {
			m.LocalQuery = ""
			m.GlobalMode = false
			m.GlobalMatches = nil
			m.Cursor = 0
			return m, nil
		}

		// dd double-tap (raw key check before the switch)
		if msg.String() == "d" {
			if m.PendingDD && time.Now().Before(m.DDDeadline) {
				m.PendingDD = false
				m.ConfirmMode = true
				m.ConfirmHard = false
				return m, nil
			}
			m.PendingDD = true
			m.DDDeadline = time.Now().Add(500 * time.Millisecond)
			return m, nil
		}
		// Any non-d key resets the pending dd state.
		m.PendingDD = false

		// Direct keybind dispatch (palette closed, prompt idle).
		if m.PromptMode == promptNone {
			if act, ok := m.ActionByKey[msg.String()]; ok {
				if sel := m.selectedNode(); sel != nil {
					res, _ := act.Run(sel)
					m.ActionOut = &res
					return m, nil
				}
			}
		}

		switch {
		case key.Matches(msg, m.Keys.Palette):
			if m.PromptMode == promptNone {
				m.PaletteMode = true
				m.Palette = palette.New(m.Theme)
				m.Palette.SetItems(m.paletteItems())
				return m, nil
			}
		case key.Matches(msg, m.Keys.Quit):
			return m, tea.Quit
		case key.Matches(msg, m.Keys.Up):
			if m.Cursor > 0 {
				m.Cursor--
			}
		case key.Matches(msg, m.Keys.Down):
			if m.Cursor < len(m.visibleSiblings())-1 {
				m.Cursor++
			}
		case key.Matches(msg, m.Keys.In):
			if m.GlobalMode {
				if len(m.GlobalMatches) > 0 && m.Cursor < len(m.GlobalMatches) {
					target := m.GlobalMatches[m.Cursor]
					m.Current = target.Parent
					for i, s := range m.Current.Children {
						if s == target {
							m.Cursor = i
							break
						}
					}
					m.GlobalMode = false
					m.GlobalMatches = nil
				}
				return m, nil
			}
			if n := m.selectedNode(); n != nil && len(n.Children) > 0 {
				m.Current = n
				m.Cursor = 0
			}
		case key.Matches(msg, m.Keys.Out):
			if m.Current.Parent != nil {
				prev := m.Current
				m.Current = m.Current.Parent
				for i, s := range m.Current.Children {
					if s == prev {
						m.Cursor = i
						break
					}
				}
			}
		case key.Matches(msg, m.Keys.Top):
			m.Cursor = 0
		case key.Matches(msg, m.Keys.Bottom):
			if n := len(m.visibleSiblings()); n > 0 {
				m.Cursor = n - 1
			} else {
				m.Cursor = 0
			}
		case key.Matches(msg, m.Keys.Refresh):
			savedCurrent := m.Current.Path
			savedSelected := ""
			if sel := m.selectedNode(); sel != nil {
				savedSelected = sel.Path
			}
			if err := m.Tree.Reload(); err == nil {
				m.restoreNav(savedCurrent, savedSelected)
			}
		case key.Matches(msg, m.Keys.ToggleDone):
			m.ShowDone = !m.ShowDone
		case key.Matches(msg, m.Keys.StatusCycle):
			if n := m.selectedNode(); n != nil {
				if err := m.Tree.SetStatus(n, n.FM.Status.Cycle()); err != nil {
					m.ActionOut = &action.Result{Stderr: err.Error(), ExitCode: 1}
				}
			}
		case key.Matches(msg, m.Keys.MoveUp):
			if n := m.selectedNode(); n != nil {
				if err := m.Tree.MoveUp(n); err == nil && m.Cursor > 0 {
					m.Cursor--
				}
			}
		case key.Matches(msg, m.Keys.MoveDown):
			if n := m.selectedNode(); n != nil {
				if err := m.Tree.MoveDown(n); err == nil && m.Cursor < len(m.Current.Children)-1 {
					m.Cursor++
				}
			}
		case key.Matches(msg, m.Keys.HardDelete):
			if m.selectedNode() != nil {
				m.ConfirmMode = true
				m.ConfirmHard = true
			}
		case key.Matches(msg, m.Keys.New):
			if m.PromptMode == promptNone {
				m.PromptMode = promptNew
				m.Prompt.SetLabel("new:")
				m.Prompt.Reset()
				m.Prompt.Focus()
				return m, nil
			}
		case key.Matches(msg, m.Keys.QuickNew):
			if m.PromptMode == promptNone {
				m.PromptMode = promptQuickNew
				m.Prompt.SetLabel("inbox:")
				m.Prompt.Reset()
				m.Prompt.Focus()
				return m, nil
			}
		case key.Matches(msg, m.Keys.Rename):
			if m.PromptMode == promptNone && m.selectedNode() != nil {
				m.PromptMode = promptRename
				m.Prompt.SetLabel("rename:")
				m.Prompt.Reset()
				m.Prompt.SetValue(m.selectedNode().Title())
				m.Prompt.Focus()
				return m, nil
			}
		case key.Matches(msg, m.Keys.MoveTo):
			if m.PromptMode == promptNone && m.selectedNode() != nil {
				m.PromptMode = promptMoveTo
				m.Prompt.SetLabel("move to:")
				m.Prompt.Reset()
				m.Prompt.Focus()
				return m, nil
			}
		case key.Matches(msg, m.Keys.Edit):
			if n := m.selectedNode(); n != nil {
				target := filepath.Join(n.Path, "index.md")
				return m, tea.ExecProcess(externalEditorCmd(target), func(err error) tea.Msg {
					return editorClosedMsg{}
				})
			}
		case key.Matches(msg, m.Keys.SearchLocal):
			if m.PromptMode == promptNone {
				m.PromptMode = promptSearchLocal
				m.Prompt.SetLabel("/")
				m.Prompt.Reset()
				m.Prompt.Focus()
				return m, nil
			}
		case key.Matches(msg, m.Keys.SearchGlobal):
			if m.PromptMode == promptNone {
				m.PromptMode = promptSearchGlobal
				m.Prompt.SetLabel("?")
				m.Prompt.Reset()
				m.Prompt.Focus()
				return m, nil
			}
		}
	}
	return m, nil
}

func (m *Model) submitPrompt() tea.Cmd {
	val := strings.TrimSpace(m.Prompt.Value())
	switch m.PromptMode {
	case promptNew:
		if val != "" {
			if _, err := m.Tree.Create(m.Current, val); err == nil {
				m.Cursor = len(m.Current.Children) - 1
			} else {
				m.ActionOut = &action.Result{Stderr: err.Error(), ExitCode: 1}
			}
		}
	case promptQuickNew:
		if val != "" {
			target := m.inboxNode()
			if target != nil {
				if _, err := m.Tree.Create(target, val); err != nil {
					m.ActionOut = &action.Result{Stderr: err.Error(), ExitCode: 1}
				}
			}
		}
	case promptRename:
		if val != "" && m.selectedNode() != nil {
			if err := m.Tree.Rename(m.selectedNode(), val); err != nil {
				m.ActionOut = &action.Result{Stderr: err.Error(), ExitCode: 1}
			}
		}
	case promptMoveTo:
		if val != "" && m.selectedNode() != nil {
			candidates := m.Tree.AllNodes()
			labels := make([]string, len(candidates))
			for i, c := range candidates {
				labels[i] = c.Title()
			}
			matches := fuzzy.Find(val, labels)
			if len(matches) > 0 {
				target := candidates[matches[0].Index]
				if err := m.Tree.MoveTo(m.selectedNode(), target); err != nil {
					m.ActionOut = &action.Result{Stderr: err.Error(), ExitCode: 1}
				}
			}
		}
	case promptSearchLocal:
		m.LocalQuery = val
		m.Cursor = 0
	case promptSearchGlobal:
		m.GlobalMatches = search.GlobalFuzzy(m.Tree.AllNodes(), val)
		m.GlobalMode = true
		m.Cursor = 0
	}
	m.PromptMode = promptNone
	m.Prompt.Blur()
	return nil
}

func (m *Model) inboxNode() *model.Node {
	inboxName := strings.TrimSpace(m.Cfg.Inbox)
	if inboxName == "" {
		inboxName = "999-inbox"
	}
	// Strip the numeric prefix if present so we compare by slug.
	_, slug, ok := store.ParsePrefix(inboxName)
	if !ok {
		slug = inboxName // user gave a bare slug like "inbox"
	}
	for _, c := range m.Tree.Root.Children {
		if c.Slug == slug {
			return c
		}
	}
	// Not found — create a top-level inbox category. Use the slug as title;
	// its on-disk prefix will be assigned by Tree.Create (next gap-of-10).
	n, err := m.Tree.Create(m.Tree.Root, slug)
	if err != nil {
		return nil
	}
	return n
}

func (m *Model) paletteItems() []palette.Item {
	items := []palette.Item{
		{Name: "refresh", Run: func() tea.Cmd {
			_ = m.Tree.Reload()
			m.PaletteMode = false
			return nil
		}},
	}
	sel := m.selectedNode()
	if sel != nil {
		for _, a := range m.ActionReg.Applicable(sel) {
			act := a
			items = append(items, palette.Item{
				Name: act.Name,
				Hint: act.Def.Cmd,
				Run: func() tea.Cmd {
					res, _ := act.Run(sel)
					m.ActionOut = &res
					m.PaletteMode = false
					return nil
				},
			})
		}
	}
	return items
}

func externalEditorCmd(path string) *exec.Cmd {
	line := editor.EditorCmd()
	parts := strings.Fields(line)
	args := append(parts[1:], path)
	return exec.Command(parts[0], args...)
}

func (m *Model) View() string {
	border := lipgloss.NewStyle().Foreground(m.Theme.Palette.Border)
	head := m.Breadcrumb.View(m.Current)
	side := m.Sidebar.View(m.visibleSiblings(), m.Cursor, m.ShowDone)
	det := m.Details.View(m.selectedNode())
	pane := lipgloss.JoinHorizontal(lipgloss.Top, side, border.Render(" │ "), det)
	if m.ConfirmMode {
		confirmMsg := "soft-delete?"
		if m.ConfirmHard {
			confirmMsg = "HARD DELETE? (cannot be undone)"
		}
		bar := lipgloss.NewStyle().
			Foreground(m.Theme.Status.Blocked).
			Bold(true).
			Render(confirmMsg + " [y/N]")
		return head + "\n" + pane + "\n" + bar
	}
	if m.PromptMode != promptNone {
		return head + "\n" + pane + "\n" + m.Prompt.View()
	}
	if m.PaletteMode {
		return head + "\n" + pane + "\n" + m.Palette.View()
	}
	if m.ActionOut != nil {
		out := fmt.Sprintf("exit=%d\n%s\n%s",
			m.ActionOut.ExitCode, m.ActionOut.Stdout, m.ActionOut.Stderr)
		return head + "\n" + pane + "\n" +
			lipgloss.NewStyle().Foreground(m.Theme.Palette.Dim).Render(out)
	}
	return head + "\n" + pane
}
