package prompt

import (
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/mwdomino/cascade/internal/theme"
)

type Model struct {
	Theme *theme.Theme
	Label string
	ti    textinput.Model
}

func New(th *theme.Theme) Model {
	ti := textinput.New()
	ti.Prompt = ""
	return Model{Theme: th, ti: ti}
}

func (m Model) Focused() bool      { return m.ti.Focused() }
func (m *Model) Focus()            { m.ti.Focus() }
func (m *Model) Blur()             { m.ti.Blur() }
func (m *Model) Reset()            { m.ti.Reset() }
func (m *Model) SetLabel(s string) { m.Label = s }
func (m *Model) SetValue(s string) { m.ti.SetValue(s) }
func (m Model) Value() string      { return m.ti.Value() }

func (m Model) Update(msg tea.Msg) (Model, tea.Cmd) {
	var cmd tea.Cmd
	m.ti, cmd = m.ti.Update(msg)
	return m, cmd
}

func (m Model) View() string {
	if !m.ti.Focused() {
		return ""
	}
	label := lipgloss.NewStyle().Foreground(m.Theme.Palette.Accent).Render(m.Label)
	return label + " " + m.ti.View()
}
