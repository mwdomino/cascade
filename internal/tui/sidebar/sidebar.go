package sidebar

import (
	"fmt"
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

func (m Model) View(items []*model.Node, cursor int, showDone bool) string {
	if len(items) == 0 {
		return lipgloss.NewStyle().Foreground(m.Theme.Palette.Dim).Render("(empty)")
	}
	var b strings.Builder
	for i, it := range items {
		row := m.renderRow(it, i == cursor)
		b.WriteString(row)
		b.WriteString("\n")
	}
	return b.String()
}

func (m Model) renderRow(n *model.Node, selected bool) string {
	glyph := m.Theme.StatusGlyph(n.FM.Status)
	title := n.Title()
	progress := ""
	if done, total := n.ProgressDoneTotal(); total > 0 {
		progress = lipgloss.NewStyle().
			Foreground(m.Theme.Palette.Dim).
			Render(fmt.Sprintf("  [%d/%d]", done, total))
	}
	dim := n.FM.Status == model.StatusDone
	titleStyle := lipgloss.NewStyle().Foreground(m.Theme.Palette.Fg)
	if dim {
		titleStyle = titleStyle.Foreground(m.Theme.Palette.Dim).Strikethrough(true)
	}
	rendered := glyph + " " + titleStyle.Render(title) + progress
	if selected {
		return lipgloss.NewStyle().
			Background(m.Theme.Selection.CursorBg).
			Render("> " + rendered)
	}
	return "  " + rendered
}
