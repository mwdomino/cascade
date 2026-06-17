package palette

import (
	"strings"
	"testing"

	"github.com/mwdomino/cascade/internal/config"
	"github.com/mwdomino/cascade/internal/theme"
)

func TestPaletteFiltering(t *testing.T) {
	th, _ := theme.Resolve(&config.Config{ThemeName: "dracula"})
	m := New(th)
	m.SetItems([]Item{
		{Name: "create-github-issue"},
		{Name: "archive"},
		{Name: "open-in-editor"},
	})
	m.ti.SetValue("git")
	out := m.View()
	if !strings.Contains(out, "create-github-issue") {
		t.Errorf("expected github match in output:\n%s", out)
	}
	if strings.Contains(out, "archive") {
		t.Errorf("archive should not match 'git':\n%s", out)
	}
}
