package details

import (
	"fmt"
	"strings"
	"testing"

	"github.com/mwdomino/cascade/internal/config"
	"github.com/mwdomino/cascade/internal/model"
	"github.com/mwdomino/cascade/internal/theme"
)

func TestScrollClipsAndIndicates(t *testing.T) {
	th, _ := theme.Resolve(&config.Config{ThemeName: "dracula"})
	// Use 100 children — the Subtasks block renders outside glamour, one
	// rendered line per child, so the math is predictable.
	n := &model.Node{FM: model.Frontmatter{Title: "Big"}}
	for i := 0; i < 100; i++ {
		n.Children = append(n.Children, &model.Node{
			FM:     model.Frontmatter{Title: fmt.Sprintf("child %d", i), Status: model.StatusTodo},
			Parent: n,
		})
	}
	m := Model{Theme: th, Width: 80, Height: 10}

	out := m.View(n)
	if !strings.Contains(out, "↓ more") {
		t.Errorf("expected more-indicator when content overflows:\n%s", out)
	}
	if !strings.Contains(out, "child 0") {
		t.Errorf("expected first child visible before scroll:\n%s", out)
	}

	m.ScrollDown(50)
	out = m.View(n)
	if strings.Contains(out, "child 0") {
		t.Errorf("first child should be scrolled past:\n%s", out)
	}

	// Scroll way past the end. View must clamp YOffset.
	m.ScrollDown(10_000)
	out = m.View(n)
	if strings.Contains(out, "↓ more") {
		t.Errorf("expected no more-indicator at the bottom:\n%s", out)
	}
	if !strings.Contains(out, "child 99") {
		t.Errorf("expected last child visible at end:\n%s", out)
	}
}

func TestScrollResetsOnSelectionChange(t *testing.T) {
	th, _ := theme.Resolve(&config.Config{ThemeName: "dracula"})
	m := Model{Theme: th, Width: 80, Height: 10}
	a := &model.Node{Path: "/a", FM: model.Frontmatter{Title: "A"}, Body: strings.Repeat("x\n", 100)}
	b := &model.Node{Path: "/b", FM: model.Frontmatter{Title: "B"}, Body: "small"}
	_ = m.View(a)
	m.ScrollDown(20)
	if m.YOffset == 0 {
		t.Fatal("scroll not applied")
	}
	_ = m.View(b)
	if m.YOffset != 0 {
		t.Errorf("YOffset should reset on selection change, got %d", m.YOffset)
	}
}

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
