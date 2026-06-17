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

func TestDoneItemRenderedWithShowDone(t *testing.T) {
	th, _ := theme.Resolve(&config.Config{ThemeName: "dracula"})
	m := Model{Theme: th, Width: 40}
	// When the caller passes a done item (showDone=true), it should be rendered
	// with dim/strikethrough styling (sidebar renders exactly what it's given).
	items := []*model.Node{
		mkNode("a", model.StatusTodo),
		mkNode("b", model.StatusDone),
		mkNode("c", model.StatusDoing),
	}
	out := m.View(items, 0, true, false)
	if !strings.Contains(out, "b") {
		t.Errorf("done item should be rendered when passed to View:\n%s", out)
	}
}

func TestSidebarRendersExactItems(t *testing.T) {
	th, _ := theme.Resolve(&config.Config{ThemeName: "dracula"})
	m := Model{Theme: th, Width: 40}
	// Filter happens upstream; sidebar renders exactly what it receives.
	items := []*model.Node{
		mkNode("a", model.StatusTodo),
		mkNode("c", model.StatusDoing),
	}
	out := m.View(items, 0, false, false)
	if !strings.Contains(out, "a") || !strings.Contains(out, "c") {
		t.Errorf("expected both items rendered:\n%s", out)
	}
}

func TestDotDotRendersAtTopWhenRequested(t *testing.T) {
	th, _ := theme.Resolve(&config.Config{ThemeName: "dracula"})
	m := Model{Theme: th, Width: 40}
	items := []*model.Node{mkNode("a", model.StatusTodo)}
	out := m.View(items, 0, false, true)
	if !strings.Contains(out, "..") {
		t.Errorf("expected `..` row when showDotDot=true:\n%s", out)
	}
	// cursor=0 should highlight `..`, NOT the child.
	firstLine := strings.SplitN(out, "\n", 2)[0]
	if !strings.Contains(firstLine, "..") || !strings.Contains(firstLine, ">") {
		t.Errorf("expected cursor on `..` row:\n%s", firstLine)
	}
}

func TestEmptyHintGuidesUser(t *testing.T) {
	th, _ := theme.Resolve(&config.Config{ThemeName: "dracula"})
	m := Model{Theme: th, Width: 40}
	out := m.View(nil, 0, false, false)
	for _, want := range []string{"no tasks here yet", "n", " to add one", ":"} {
		if !strings.Contains(out, want) {
			t.Errorf("empty hint missing %q:\n%s", want, out)
		}
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
	out := m.View([]*model.Node{parent}, 0, false, false)
	if !strings.Contains(out, "[2/3]") {
		t.Errorf("expected [2/3] in output:\n%s", out)
	}
}
