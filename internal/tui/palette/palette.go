package palette

import (
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/mwdomino/cascade/internal/theme"
	"github.com/sahilm/fuzzy"
)

type Item struct {
	Name string
	Hint string
	Run  func() tea.Cmd
}

type Model struct {
	Theme *theme.Theme
	ti    textinput.Model
	Items []Item
	Cur   int
}

func New(th *theme.Theme) Model {
	ti := textinput.New()
	ti.Prompt = ": "
	ti.Focus()
	return Model{Theme: th, ti: ti}
}

func (m *Model) SetItems(items []Item) { m.Items = items; m.Cur = 0 }

func (m Model) filtered() []Item {
	q := strings.TrimSpace(m.ti.Value())
	if q == "" {
		return m.Items
	}
	names := make([]string, len(m.Items))
	for i, it := range m.Items {
		names[i] = it.Name
	}
	out := make([]Item, 0, len(m.Items))
	for _, mt := range fuzzy.Find(q, names) {
		out = append(out, m.Items[mt.Index])
	}
	return out
}

func (m Model) Update(msg tea.Msg) (Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "up":
			if m.Cur > 0 {
				m.Cur--
			}
			return m, nil
		case "down":
			if m.Cur < len(m.filtered())-1 {
				m.Cur++
			}
			return m, nil
		case "enter":
			fil := m.filtered()
			if m.Cur < len(fil) && fil[m.Cur].Run != nil {
				return m, fil[m.Cur].Run()
			}
			return m, nil
		}
	}
	var cmd tea.Cmd
	m.ti, cmd = m.ti.Update(msg)
	// After the textinput updates, clamp Cur to the new filtered list length.
	if n := len(m.filtered()); n == 0 {
		m.Cur = 0
	} else if m.Cur >= n {
		m.Cur = n - 1
	}
	return m, cmd
}

func (m Model) View() string {
	var b strings.Builder
	b.WriteString(m.ti.View())
	b.WriteString("\n")
	for i, it := range m.filtered() {
		row := it.Name
		if it.Hint != "" {
			row += "  " + lipgloss.NewStyle().Foreground(m.Theme.Palette.Dim).Render(it.Hint)
		}
		if i == m.Cur {
			row = lipgloss.NewStyle().Background(m.Theme.Selection.CursorBg).Render("> " + row)
		} else {
			row = "  " + row
		}
		b.WriteString(row)
		b.WriteString("\n")
	}
	return b.String()
}
