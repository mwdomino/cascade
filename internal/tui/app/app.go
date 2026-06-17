package app

import (
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/mwdomino/cascade/internal/config"
	"github.com/mwdomino/cascade/internal/editor"
	"github.com/mwdomino/cascade/internal/model"
	"github.com/mwdomino/cascade/internal/store"
	"github.com/mwdomino/cascade/internal/theme"
	"github.com/mwdomino/cascade/internal/tui/breadcrumb"
	"github.com/mwdomino/cascade/internal/tui/details"
	"github.com/mwdomino/cascade/internal/tui/keys"
	"github.com/mwdomino/cascade/internal/tui/prompt"
	"github.com/mwdomino/cascade/internal/tui/sidebar"
	"github.com/sahilm/fuzzy"
)

type promptMode int

const (
	promptNone     promptMode = iota
	promptNew
	promptQuickNew
	promptRename
	promptMoveTo
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

	// Confirm overlay state
	ConfirmMode bool
	ConfirmHard bool
	PendingDD   bool
	DDDeadline  time.Time

	Width, Height int
}

func New(tree *store.Tree, th *theme.Theme, cfg *config.Config) tea.Model {
	return &Model{
		Tree:       tree,
		Theme:      th,
		Cfg:        cfg,
		Keys:       keys.Default(),
		Current:    tree.Root,
		Sidebar:    sidebar.Model{Theme: th},
		Details:    details.Model{Theme: th},
		Breadcrumb: breadcrumb.Model{Theme: th},
		Prompt:     prompt.New(th),
	}
}

func (m *Model) Init() tea.Cmd { return nil }

func (m *Model) visibleSiblings() []*model.Node { return m.Current.Children }

func (m *Model) selectedNode() *model.Node {
	sibs := m.visibleSiblings()
	if len(sibs) == 0 || m.Cursor < 0 || m.Cursor >= len(sibs) {
		return nil
	}
	return sibs[m.Cursor]
}

func (m *Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case editorClosedMsg:
		_ = m.Tree.Reload()
		m.Current = m.Tree.Root
		m.Cursor = 0
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

		switch {
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
			if err := m.Tree.Reload(); err == nil {
				m.Current = m.Tree.Root
				m.Cursor = 0
			}
		case key.Matches(msg, m.Keys.ToggleDone):
			m.ShowDone = !m.ShowDone
		case key.Matches(msg, m.Keys.StatusCycle):
			if n := m.selectedNode(); n != nil {
				m.Tree.SetStatus(n, n.FM.Status.Cycle())
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
			}
		}
	case promptQuickNew:
		if val != "" {
			target := m.inboxNode()
			if target != nil {
				m.Tree.Create(target, val)
			}
		}
	case promptRename:
		if val != "" && m.selectedNode() != nil {
			m.Tree.Rename(m.selectedNode(), val)
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
				m.Tree.MoveTo(m.selectedNode(), target)
			}
		}
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
	return head + "\n" + pane
}
