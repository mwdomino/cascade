package sidebar

import (
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/mwdomino/cascade/internal/model"
	"github.com/mwdomino/cascade/internal/theme"
)

type Model struct {
	Theme  *theme.Theme
	Width  int
	Height int
}

func (m Model) View(items []*model.Node, cursor int) string {
	if len(items) == 0 {
		return lipgloss.NewStyle().
			Foreground(m.Theme.Palette.Dim).
			Render("(empty)")
	}
	var b strings.Builder
	for i, it := range items {
		row := it.Title()
		if i == cursor {
			row = lipgloss.NewStyle().
				Background(m.Theme.Selection.CursorBg).
				Foreground(m.Theme.Palette.Fg).
				Render("> " + row)
		} else {
			row = "  " + row
		}
		b.WriteString(row)
		b.WriteString("\n")
	}
	return b.String()
}
