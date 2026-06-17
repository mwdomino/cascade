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

func (m Model) View(items []*model.Node, cursor int, showDone bool, showDotDot bool) string {
	var b strings.Builder
	offset := 0
	if showDotDot {
		b.WriteString(m.renderDotDot(cursor == 0))
		b.WriteString("\n")
		offset = 1
	}
	if len(items) == 0 {
		if showDotDot {
			return b.String() + m.emptyHint()
		}
		return m.emptyHint()
	}
	for i, it := range items {
		row := m.renderRow(it, i+offset == cursor)
		b.WriteString(row)
		b.WriteString("\n")
	}
	return b.String()
}

func (m Model) renderDotDot(selected bool) string {
	text := lipgloss.NewStyle().
		Foreground(m.Theme.Palette.Dim).
		Render("..  (go up)")
	if selected {
		return lipgloss.NewStyle().
			Background(m.Theme.Selection.CursorBg).
			Render("> " + text)
	}
	return "  " + text
}

func (m Model) emptyHint() string {
	dim := lipgloss.NewStyle().Foreground(m.Theme.Palette.Dim)
	accent := lipgloss.NewStyle().Foreground(m.Theme.Palette.Accent).Bold(true)
	lines := []string{
		dim.Render("no tasks here yet"),
		"",
		dim.Render("press ") + accent.Render("n") + dim.Render(" to add one"),
		dim.Render("press ") + accent.Render(":") + dim.Render(" for the command palette"),
		dim.Render("press ") + accent.Render("h") + dim.Render(" to go back"),
	}
	return strings.Join(lines, "\n")
}

func (m Model) renderRow(n *model.Node, selected bool) string {
	glyph := m.Theme.NodeGlyph(n)
	title := n.Title()
	progress := ""
	if done, total := n.ProgressDoneTotal(); total > 0 {
		progress = lipgloss.NewStyle().
			Foreground(m.Theme.Palette.Dim).
			Render(fmt.Sprintf("  [%d/%d]", done, total))
	}
	// Only tasks get dim+strike when done; container types stay legible.
	dim := n.EffectiveType() == model.TypeTask && n.FM.Status == model.StatusDone
	var titleStyle lipgloss.Style
	switch n.EffectiveType() {
	case model.TypeProject:
		titleStyle = lipgloss.NewStyle().Foreground(m.Theme.Palette.Accent).Bold(true)
	case model.TypeFolder:
		titleStyle = lipgloss.NewStyle().Foreground(m.Theme.Palette.Fg).Bold(true)
	default:
		titleStyle = lipgloss.NewStyle().Foreground(m.Theme.Palette.Fg)
	}
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
