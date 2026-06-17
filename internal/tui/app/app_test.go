package app

import (
	"path/filepath"
	"strings"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/x/exp/teatest"
	"github.com/mwdomino/cascade/internal/action"
	"github.com/mwdomino/cascade/internal/config"
	"github.com/mwdomino/cascade/internal/model"
	"github.com/mwdomino/cascade/internal/store"
	"github.com/mwdomino/cascade/internal/theme"
	"github.com/mwdomino/cascade/internal/tui/keys"
)

func setup(t *testing.T) (*store.Tree, *theme.Theme, *config.Config) {
	t.Helper()
	dir := t.TempDir()
	tree, _ := store.Load(dir)
	work, _ := tree.Create(tree.Root, "Work")
	tree.Create(work, "Ship v1")
	tree.Create(work, "Fix bug")
	tree.Create(tree.Root, "Personal")
	th, _ := theme.Resolve(&config.Config{ThemeName: "dracula"})
	_ = filepath.Join // silence unused on some builds
	_ = model.StatusTodo
	return tree, th, &config.Config{}
}

func newModel(tree *store.Tree, th *theme.Theme, cfg *config.Config) tea.Model {
	return New(tree, th, cfg, action.NewRegistry(nil))
}

func TestDrillInAndBack(t *testing.T) {
	tree, th, cfg := setup(t)
	tm := teatest.NewTestModel(t, newModel(tree, th, cfg),
		teatest.WithInitialTermSize(120, 40))

	tm.Send(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'l'}}) // drill into "Work"
	tm.Send(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'j'}}) // cursor → Fix bug
	tm.Send(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'h'}}) // back to root
	tm.Send(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'q'}})

	tm.WaitFinished(t, teatest.WithFinalTimeout(2*time.Second))
}

func TestBottomOnEmpty(t *testing.T) {
	dir := t.TempDir()
	tree, _ := store.Load(dir)
	th, _ := theme.Resolve(&config.Config{ThemeName: "dracula"})
	tm := teatest.NewTestModel(t, newModel(tree, th, &config.Config{}),
		teatest.WithInitialTermSize(120, 40))

	tm.Send(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'G'}}) // Bottom on empty list
	tm.Send(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'q'}})

	tm.WaitFinished(t, teatest.WithFinalTimeout(2*time.Second))
}

func TestToggleShowDone(t *testing.T) {
	tree, th, cfg := setup(t)
	// Mark "Personal" as done
	for _, n := range tree.Root.Children {
		if n.Slug == "personal" {
			tree.SetStatus(n, model.StatusDone)
		}
	}
	tm := teatest.NewTestModel(t, newModel(tree, th, cfg),
		teatest.WithInitialTermSize(120, 40))
	tm.Send(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'Z'}})
	tm.Send(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'q'}})
	tm.WaitFinished(t, teatest.WithFinalTimeout(2*time.Second))
}

func TestHelpToggle(t *testing.T) {
	tree, th, cfg := setup(t)
	m := newModel(tree, th, cfg).(*Model)
	// Press '?' → HelpMode on.
	m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'?'}})
	if !m.HelpMode {
		t.Fatal("? should open HelpMode")
	}
	if !strings.Contains(m.View(), "cascade — keybindings") {
		t.Errorf("help overlay missing title")
	}
	// Any key closes it.
	m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'x'}})
	if m.HelpMode {
		t.Error("any key should close HelpMode")
	}
}

func TestEnterOnDotDotAscends(t *testing.T) {
	tree, th, cfg := setup(t)
	// setup creates "work" (with children "ship-v1", "fix-bug") and "personal".
	m := newModel(tree, th, cfg).(*Model)
	m.Update(tea.WindowSizeMsg{Width: 120, Height: 40})
	// Drill into "work" (first root child).
	m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'l'}})
	if m.Current.Slug != "work" {
		t.Fatalf("expected to be inside work, got %q", m.Current.Slug)
	}
	// Cursor should be on the first real child after drilling in.
	if m.cursorIsDotDot() {
		t.Errorf("cursor unexpectedly on `..` right after drilling in")
	}
	// gg jumps to top → that's `..`.
	m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'g'}})
	if !m.cursorIsDotDot() {
		t.Errorf("gg should land cursor on `..`")
	}
	// Enter on `..` goes up.
	m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'l'}})
	if m.Current != tree.Root {
		t.Errorf("Enter on `..` should ascend to root, got %q", m.Current.Slug)
	}
	if m.selectedNode() == nil || m.selectedNode().Slug != "work" {
		t.Errorf("after ascend, cursor should land on 'work', got %v", m.selectedNode())
	}
}

func TestHintBarDefault(t *testing.T) {
	tree, th, cfg := setup(t)
	m := newModel(tree, th, cfg).(*Model)
	// Send a window-size msg so dimensions exist.
	m.Update(tea.WindowSizeMsg{Width: 120, Height: 40})
	out := m.View()
	for _, want := range []string{"n", "new", "drill in", "help", "actions"} {
		if !strings.Contains(out, want) {
			t.Errorf("hint bar missing %q in default view", want)
		}
	}
}

func TestStatusCycle(t *testing.T) {
	tree, th, cfg := setup(t)
	tm := teatest.NewTestModel(t, newModel(tree, th, cfg),
		teatest.WithInitialTermSize(120, 40))
	tm.Send(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'x'}}) // todo -> doing
	tm.Send(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'q'}})
	tm.WaitFinished(t, teatest.WithFinalTimeout(2*time.Second))
	first := tree.Root.Children[0]
	if first.FM.Status != model.StatusDoing {
		t.Errorf("status = %q, want doing", first.FM.Status)
	}
}

func TestReorderKJ(t *testing.T) {
	tree, th, cfg := setup(t)
	tm := teatest.NewTestModel(t, newModel(tree, th, cfg),
		teatest.WithInitialTermSize(120, 40))
	tm.Send(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'j'}}) // cursor → Personal
	tm.Send(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'K'}}) // swap with Work
	tm.Send(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'q'}})
	tm.WaitFinished(t, teatest.WithFinalTimeout(2*time.Second))
	if tree.Root.Children[0].Slug != "personal" {
		t.Errorf("after reorder first = %q", tree.Root.Children[0].Slug)
	}
}

func TestSoftDeleteFlow(t *testing.T) {
	tree, th, cfg := setup(t)
	tm := teatest.NewTestModel(t, newModel(tree, th, cfg),
		teatest.WithInitialTermSize(120, 40))
	tm.Send(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'d'}})
	tm.Send(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'d'}})
	tm.Send(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'y'}})
	tm.Send(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'q'}})
	tm.WaitFinished(t, teatest.WithFinalTimeout(2*time.Second))
	if len(tree.Root.Children) != 1 {
		t.Errorf("expected 1 sibling, got %d", len(tree.Root.Children))
	}
}

func TestLocalSearch(t *testing.T) {
	tree, th, cfg := setup(t)
	tm := teatest.NewTestModel(t, newModel(tree, th, cfg),
		teatest.WithInitialTermSize(120, 40))
	tm.Send(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'/'}})
	tm.Type("personal")
	tm.Send(tea.KeyMsg{Type: tea.KeyEnter})
	tm.Send(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'q'}})
	tm.WaitFinished(t, teatest.WithFinalTimeout(2*time.Second))
}

func TestPaletteRefresh(t *testing.T) {
	tree, th, cfg := setup(t)
	reg := action.NewRegistry(nil)
	tm := teatest.NewTestModel(t, New(tree, th, cfg, reg),
		teatest.WithInitialTermSize(120, 40))
	tm.Send(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{':'}})
	tm.Type("refresh")
	tm.Send(tea.KeyMsg{Type: tea.KeyEnter})
	tm.Send(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'q'}})
	tm.WaitFinished(t, teatest.WithFinalTimeout(2*time.Second))
}

func TestZHideDoneStableCursor(t *testing.T) {
	dir := t.TempDir()
	tree, _ := store.Load(dir)
	tree.Create(tree.Root, "Alpha")
	beta, _ := tree.Create(tree.Root, "Beta")
	tree.SetStatus(beta, model.StatusDone)
	tree.Create(tree.Root, "Gamma")
	th, _ := theme.Resolve(&config.Config{ThemeName: "dracula"})
	m := &Model{
		Tree:     tree,
		Theme:    th,
		Cfg:      &config.Config{},
		Keys:     keys.Default(),
		Current:  tree.Root,
		ShowDone: false,
	}
	// Send j — should advance past hidden Beta to Gamma
	m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'j'}})
	got := m.selectedNode()
	if got == nil {
		t.Fatal("selectedNode is nil after j")
	}
	if got.Slug != "gamma" {
		t.Errorf("cursor should land on gamma, got %q", got.Slug)
	}
	if got.Slug == "beta" {
		t.Errorf("cursor must not land on hidden done item beta")
	}
}

func TestCaptureNewTask(t *testing.T) {
	tree, th, cfg := setup(t)
	tm := teatest.NewTestModel(t, newModel(tree, th, cfg),
		teatest.WithInitialTermSize(120, 40))
	tm.Send(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'n'}})
	tm.Type("Brand New Task")
	tm.Send(tea.KeyMsg{Type: tea.KeyEnter})
	tm.Send(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'q'}})
	tm.WaitFinished(t, teatest.WithFinalTimeout(2*time.Second))

	found := false
	for _, c := range tree.Root.Children {
		if c.Slug == "brand-new-task" {
			found = true
			break
		}
	}
	if !found {
		t.Error("new task not created on disk")
	}
}
