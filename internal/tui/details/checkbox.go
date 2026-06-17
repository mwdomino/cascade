package details

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

// checkboxRe matches a markdown task-list line: optional leading whitespace,
// a `-` or `*` bullet, the box (` `, `x`, or `X`), and the task text.
var checkboxRe = regexp.MustCompile(`^(\s*[-*]\s)\[([ xX])\](\s*)(.*)$`)

// CountCheckboxes returns the number of `- [ ]` / `- [x]` lines in body.
func CountCheckboxes(body string) int {
	n := 0
	for _, l := range strings.Split(body, "\n") {
		if checkboxRe.MatchString(l) {
			n++
		}
	}
	return n
}

// ToggleCheckbox flips the state of the labeled checkbox (1-indexed) in
// body and returns the new body. Returns (body, false) if the label is out
// of range.
func ToggleCheckbox(body string, label int) (string, bool) {
	if label < 1 {
		return body, false
	}
	lines := strings.Split(body, "\n")
	seen := 0
	for i, l := range lines {
		mt := checkboxRe.FindStringSubmatch(l)
		if mt == nil {
			continue
		}
		seen++
		if seen != label {
			continue
		}
		ticked := strings.ToLower(mt[2]) == "x"
		box := "[x]"
		if ticked {
			box = "[ ]"
		}
		lines[i] = mt[1] + box + mt[3] + mt[4]
		return strings.Join(lines, "\n"), true
	}
	return body, false
}

// renderLabeledBody re-renders body so each checkbox line shows its 1-based
// label prefix, intended for the toggle-mode overlay. Non-checkbox lines pass
// through unchanged.
func renderLabeledBody(body string, accent, done, todo, dim lipgloss.Color) string {
	labelStyle := lipgloss.NewStyle().Foreground(accent).Bold(true)
	doneStyle := lipgloss.NewStyle().Foreground(done)
	todoStyle := lipgloss.NewStyle().Foreground(todo)
	dimStyle := lipgloss.NewStyle().Foreground(dim)

	lines := strings.Split(body, "\n")
	label := 1
	for i, l := range lines {
		mt := checkboxRe.FindStringSubmatch(l)
		if mt == nil {
			lines[i] = dimStyle.Render(l)
			continue
		}
		ticked := strings.ToLower(mt[2]) == "x"
		glyph := "○"
		gs := todoStyle
		if ticked {
			glyph = "✓"
			gs = doneStyle
		}
		labelTag := labelStyle.Render(fmt.Sprintf("[%d]", label))
		text := mt[4]
		if ticked {
			text = doneStyle.Strikethrough(true).Render(text)
		} else {
			text = lipgloss.NewStyle().Render(text)
		}
		lines[i] = "  " + labelTag + " " + gs.Render(glyph) + " " + text
		label++
	}
	return strings.Join(lines, "\n")
}
