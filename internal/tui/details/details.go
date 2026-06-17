package details

import (
	"github.com/charmbracelet/lipgloss"
	"github.com/mwdomino/cascade/internal/model"
	"github.com/mwdomino/cascade/internal/theme"
)

type Model struct {
	Theme  *theme.Theme
	Width  int
	Height int
}

func (m Model) View(n *model.Node) string {
	if n == nil {
		return lipgloss.NewStyle().Foreground(m.Theme.Palette.Dim).Render("(no selection)")
	}
	titleStyle := lipgloss.NewStyle().Foreground(m.Theme.Markdown.Heading).Bold(true)
	return titleStyle.Render("# "+n.Title()) + "\n\n" + n.Body
}
