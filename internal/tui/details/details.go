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
	Theme           *theme.Theme
	Width           int
	Height          int
	YOffset         int
	LabelCheckboxes bool // when true, body is rendered with [1]…[N] labels for the toggle overlay
	lastPath        string

	// Glamour renderer + memoization. The renderer parses style configs and
	// compiles regexes on construction; reusing it across frames is a real
	// win since View runs on every keystroke.
	renderer     *glamour.TermRenderer
	rendererW    int
	cachedBody   string
	cachedOutput string
}

func (m *Model) ScrollDown(lines int) {
	m.YOffset += lines
	// Final clamp happens in View where we know the content height.
}

func (m *Model) ScrollUp(lines int) {
	m.YOffset -= lines
	if m.YOffset < 0 {
		m.YOffset = 0
	}
}

func (m *Model) ResetScroll() { m.YOffset = 0 }

// ClearCache invalidates the glamour renderer + memoized output. Callers
// should run this when the theme changes at runtime so the next View
// rebuilds with the new colors.
func (m *Model) ClearCache() {
	m.renderer = nil
	m.rendererW = 0
	m.cachedBody = ""
	m.cachedOutput = ""
}

func (m *Model) View(n *model.Node) string {
	if n == nil {
		return lipgloss.NewStyle().Foreground(m.Theme.Palette.Dim).Render("(no selection)")
	}
	// Reset scroll when the selection changes.
	if n.Path != m.lastPath {
		m.YOffset = 0
		m.lastPath = n.Path
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

	rendered := strings.TrimRight(n.Body, "\n")
	if m.LabelCheckboxes {
		// Bypass glamour so the [1]…[N] labels and direct ANSI styling survive.
		rendered = renderLabeledBody(n.Body,
			m.Theme.Palette.Accent,
			m.Theme.Status.Done,
			m.Theme.Status.Todo,
			m.Theme.Palette.Fg,
		)
	} else if rendered != "" {
		// Rebuild the renderer only when the width changes; cache the rendered
		// output by body content so unchanged bodies reuse the previous render.
		if m.renderer == nil || m.rendererW != width {
			r, err := glamour.NewTermRenderer(
				glamour.WithStyles(m.Theme.GlamourStyle()),
				glamour.WithWordWrap(width),
			)
			if err == nil {
				m.renderer = r
				m.rendererW = width
				m.cachedBody = ""
				m.cachedOutput = ""
			}
		}
		if m.renderer != nil {
			if m.cachedOutput != "" && m.cachedBody == n.Body {
				rendered = m.cachedOutput
			} else if out, rerr := m.renderer.Render(n.Body); rerr == nil {
				rendered = strings.TrimRight(out, "\n")
				m.cachedBody = n.Body
				m.cachedOutput = rendered
			}
		}
	}

	subtasks := m.synthesizeSubtasks(n)

	// Compose the scrollable portion: body + Subtasks block. Title and rule
	// stay pinned at the top.
	var scroll strings.Builder
	if rendered != "" {
		scroll.WriteString(rendered)
	}
	if subtasks != "" {
		if scroll.Len() > 0 {
			scroll.WriteString("\n")
			scroll.WriteString(rule)
			scroll.WriteString("\n")
		}
		scroll.WriteString(subtasks)
	}
	scrollContent := scroll.String()
	scrollLines := strings.Split(scrollContent, "\n")
	if scrollContent == "" {
		scrollLines = nil
	}

	headerH := lipgloss.Height(title) + 1 /* rule */
	availH := m.Height - headerH
	if availH < 1 {
		availH = 1
	}

	maxOffset := len(scrollLines) - availH
	if maxOffset < 0 {
		maxOffset = 0
	}
	if m.YOffset > maxOffset {
		m.YOffset = maxOffset
	}

	visible := scrollLines
	if m.YOffset < len(visible) {
		visible = visible[m.YOffset:]
	} else {
		visible = nil
	}
	if len(visible) > availH {
		visible = visible[:availH]
	}

	parts := []string{title, rule}
	if len(visible) > 0 {
		parts = append(parts, strings.Join(visible, "\n"))
	}

	// Show a small "more below" indicator when there's hidden content.
	if len(scrollLines) > 0 && m.YOffset+availH < len(scrollLines) {
		indicator := lipgloss.NewStyle().
			Foreground(m.Theme.Palette.Dim).
			Italic(true).
			Render("↓ more (ctrl+d / ctrl+j / pgdn)")
		parts = append(parts, indicator)
	}

	return strings.Join(parts, "\n")
}

func (m *Model) synthesizeSubtasks(n *model.Node) string {
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
