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
	if len(parts) == 0 {
		parts = []string{"~"}
	}
	style := lipgloss.NewStyle().Foreground(m.Theme.Palette.Accent).Bold(true)
	return style.Render(strings.Join(parts, " › "))
}
