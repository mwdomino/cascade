package details

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/glamour"
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
	width := m.Width
	if width <= 0 {
		width = 80
	}

	titleStyle := lipgloss.NewStyle().
		Foreground(m.Theme.Palette.Accent).
		Bold(true).
		Padding(1, 2)
	title := titleStyle.Render(n.Title())

	rule := lipgloss.NewStyle().
		Foreground(m.Theme.Palette.Border).
		Render(strings.Repeat("─", width))

	// Render the body via glamour. Containers/empty leaves may have no body.
	rendered := strings.TrimRight(n.Body, "\n")
	if rendered != "" {
		r, err := glamour.NewTermRenderer(
			glamour.WithStyles(m.Theme.GlamourStyle()),
			glamour.WithWordWrap(width),
		)
		if err == nil {
			if out, rerr := r.Render(n.Body); rerr == nil {
				rendered = strings.TrimRight(out, "\n")
			}
		}
	}

	// Build the synthesized subtasks block AFTER glamour, so lipgloss-styled
	// glyphs (which contain raw ANSI escapes) don't get fed through markdown
	// rendering and mangled into literal escape sequences.
	subtasks := m.synthesizeSubtasks(n)

	parts := []string{title, rule}
	if rendered != "" {
		parts = append(parts, rendered)
	}
	if subtasks != "" {
		parts = append(parts, subtasks)
	}
	return strings.Join(parts, "\n")
}

func (m Model) synthesizeSubtasks(n *model.Node) string {
	if len(n.Children) == 0 {
		return ""
	}
	heading := lipgloss.NewStyle().
		Foreground(m.Theme.Markdown.HeadingH2).
		Bold(true).
		Render("Subtasks")
	progressStyle := lipgloss.NewStyle().Foreground(m.Theme.Palette.Dim)
	titleStyle := lipgloss.NewStyle().Foreground(m.Theme.Palette.Fg)
	doneStyle := titleStyle.
		Foreground(m.Theme.Palette.Dim).
		Strikethrough(true)

	var b strings.Builder
	b.WriteString("\n")
	b.WriteString(heading)
	b.WriteString("\n\n")
	for _, c := range n.Children {
		glyph := m.Theme.NodeGlyph(c)
		t := c.Title()
		if c.EffectiveType() == model.TypeTask && c.FM.Status == model.StatusDone {
			t = doneStyle.Render(t)
		} else {
			t = titleStyle.Render(t)
		}
		b.WriteString("  ")
		b.WriteString(glyph)
		b.WriteString(" ")
		b.WriteString(t)
		if d, total := c.ProgressDoneTotal(); total > 0 {
			b.WriteString(progressStyle.Render(fmt.Sprintf("  [%d/%d]", d, total)))
		}
		b.WriteString("\n")
	}
	return b.String()
}
