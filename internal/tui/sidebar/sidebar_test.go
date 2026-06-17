package sidebar

import (
	"strings"
	"testing"

	"github.com/mwdomino/cascade/internal/config"
	"github.com/mwdomino/cascade/internal/model"
	"github.com/mwdomino/cascade/internal/theme"
)

func mkNode(slug string, s model.Status, kids ...*model.Node) *model.Node {
	n := &model.Node{Slug: slug, FM: model.Frontmatter{Title: slug, Status: s}}
	for _, k := range kids {
		k.Parent = n
		n.Children = append(n.Children, k)
	}
	return n
}

func TestHideDoneByDefault(t *testing.T) {
	th, _ := theme.Resolve(&config.Config{ThemeName: "dracula"})
	m := Model{Theme: th, Width: 40}
	items := []*model.Node{
		mkNode("a", model.StatusTodo),
		mkNode("b", model.StatusDone),
		mkNode("c", model.StatusDoing),
	}
	out := m.View(items, 0, false)
	if strings.Contains(out, "b") {
		t.Errorf("done item should be hidden:\n%s", out)
	}
}

func TestProgressRollup(t *testing.T) {
	th, _ := theme.Resolve(&config.Config{ThemeName: "dracula"})
	m := Model{Theme: th, Width: 60}
	parent := mkNode("parent", model.StatusDoing,
		mkNode("k1", model.StatusDone),
		mkNode("k2", model.StatusDone),
		mkNode("k3", model.StatusTodo),
	)
	out := m.View([]*model.Node{parent}, 0, false)
	if !strings.Contains(out, "[2/3]") {
		t.Errorf("expected [2/3] in output:\n%s", out)
	}
}
