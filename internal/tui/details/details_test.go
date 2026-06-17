package details

import (
	"strings"
	"testing"

	"github.com/mwdomino/cascade/internal/config"
	"github.com/mwdomino/cascade/internal/model"
	"github.com/mwdomino/cascade/internal/theme"
)

func TestSubtasksBlockSynthesized(t *testing.T) {
	th, _ := theme.Resolve(&config.Config{ThemeName: "dracula"})
	parent := &model.Node{FM: model.Frontmatter{Title: "Parent"}, Body: "Hello\n"}
	parent.Children = []*model.Node{
		{FM: model.Frontmatter{Title: "K1", Status: model.StatusDone}, Parent: parent},
		{FM: model.Frontmatter{Title: "K2", Status: model.StatusTodo}, Parent: parent},
	}
	m := Model{Theme: th, Width: 80, Height: 40}
	out := m.View(parent)
	if !strings.Contains(out, "Subtasks") {
		t.Errorf("Subtasks block missing:\n%s", out)
	}
	if !strings.Contains(out, "K1") || !strings.Contains(out, "K2") {
		t.Errorf("child titles missing:\n%s", out)
	}
}
