package app

import (
	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/mwdomino/cascade/internal/config"
	"github.com/mwdomino/cascade/internal/model"
	"github.com/mwdomino/cascade/internal/store"
	"github.com/mwdomino/cascade/internal/theme"
	"github.com/mwdomino/cascade/internal/tui/breadcrumb"
	"github.com/mwdomino/cascade/internal/tui/details"
	"github.com/mwdomino/cascade/internal/tui/keys"
	"github.com/mwdomino/cascade/internal/tui/sidebar"
)

type Model struct {
	Tree    *store.Tree
	Theme   *theme.Theme
	Cfg     *config.Config
	Keys    keys.Map
	Current *model.Node // parent of the displayed siblings
	Cursor  int

	Sidebar    sidebar.Model
	Details    details.Model
	Breadcrumb breadcrumb.Model

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
	}
}

func (m *Model) Init() tea.Cmd { return nil }

func (m *Model) visibleSiblings() []*model.Node { return m.Current.Children }

func (m *Model) selectedNode() *model.Node {
	sibs := m.visibleSiblings()
	if len(sibs) == 0 || m.Cursor >= len(sibs) {
		return nil
	}
	return sibs[m.Cursor]
}

func (m *Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
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
			m.Cursor = len(m.visibleSiblings()) - 1
		case key.Matches(msg, m.Keys.Refresh):
			if err := m.Tree.Reload(); err == nil {
				m.Current = m.Tree.Root
				m.Cursor = 0
			}
		}
	}
	return m, nil
}

func (m *Model) View() string {
	border := lipgloss.NewStyle().Foreground(m.Theme.Palette.Border)
	head := m.Breadcrumb.View(m.Current)
	side := m.Sidebar.View(m.visibleSiblings(), m.Cursor)
	det := m.Details.View(m.selectedNode())
	pane := lipgloss.JoinHorizontal(lipgloss.Top, side, border.Render(" │ "), det)
	return head + "\n" + pane
}
