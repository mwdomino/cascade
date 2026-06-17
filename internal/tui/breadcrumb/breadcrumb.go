package breadcrumb

import (
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/mwdomino/cascade/internal/model"
	"github.com/mwdomino/cascade/internal/theme"
)

type Model struct {
	Theme *theme.Theme
}

func (m Model) View(node *model.Node) string {
	var parts []string
	for cur := node; cur != nil && !cur.IsRoot(); cur = cur.Parent {
		parts = append([]string{cur.Title()}, parts...)
	}
	app := lipgloss.NewStyle().
		Foreground(m.Theme.Palette.Dim).
		Bold(true).
		Render("cascade")
	if len(parts) == 0 {
		return app
	}
	sep := lipgloss.NewStyle().Foreground(m.Theme.Palette.Dim).Render(" › ")
	path := lipgloss.NewStyle().
		Foreground(m.Theme.Palette.Accent).
		Bold(true).
		Render(strings.Join(parts, " › "))
	return app + sep + path
}
