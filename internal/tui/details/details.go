package details

import (
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
	body := "# " + n.Title() + "\n\n" + n.Body
	if len(n.Children) > 0 {
		body += "\n\n## Subtasks\n\n" + m.synthesizeChildren(n)
	}
	style := m.Theme.GlamourStyle()
	r, err := glamour.NewTermRenderer(
		glamour.WithStyles(style),
		glamour.WithWordWrap(width),
	)
	if err != nil {
		return body
	}
	out, err := r.Render(body)
	if err != nil {
		return body
	}
	return strings.TrimRight(out, "\n")
}

func (m Model) synthesizeChildren(n *model.Node) string {
	var b strings.Builder
	for _, c := range n.Children {
		b.WriteString("- ")
		b.WriteString(m.Theme.StatusGlyph(c.FM.Status))
		b.WriteString(" ")
		b.WriteString(c.Title())
		if d, total := c.ProgressDoneTotal(); total > 0 {
			b.WriteString(" `[")
			b.WriteString(itoa(d))
			b.WriteString("/")
			b.WriteString(itoa(total))
			b.WriteString("]`")
		}
		b.WriteString("\n")
	}
	return b.String()
}

func itoa(i int) string {
	const digits = "0123456789"
	if i == 0 {
		return "0"
	}
	var buf [20]byte
	pos := len(buf)
	for i > 0 {
		pos--
		buf[pos] = digits[i%10]
		i /= 10
	}
	return string(buf[pos:])
}
