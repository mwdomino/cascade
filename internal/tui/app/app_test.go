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

func TestGChordTopAndCancel(t *testing.T) {
	tree, th, cfg := setup(t)
	m := newModel(tree, th, cfg).(*Model)
	m.Update(tea.WindowSizeMsg{Width: 120, Height: 40})
	// Single g must not move the cursor; the chord is still pending.
	m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'j'}})
	beforeCursor := m.Cursor
	m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'g'}})
	if m.Cursor != beforeCursor {
		t.Errorf("single g should not move cursor; got %d want %d", m.Cursor, beforeCursor)
	}
	if !m.PendingG {
		t.Error("PendingG should be set after first g")
	}
	// Second g completes the chord.
	m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'g'}})
	if m.Cursor != 0 {
		t.Errorf("gg should move cursor to top; got %d", m.Cursor)
	}
	if m.PendingG {
		t.Error("PendingG should clear after completion")
	}
	// A non-g, non-n key cancels the chord and dispatches normally.
	m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'g'}})
	m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'j'}})
	if m.PendingG {
		t.Error("PendingG should clear after non-chord key")
	}
}

func TestGChordQuickNew(t *testing.T) {
	tree, th, cfg := setup(t)
	m := newModel(tree, th, cfg).(*Model)
	m.Update(tea.WindowSizeMsg{Width: 120, Height: 40})
	// gn opens the quick-capture prompt with label "inbox:".
	m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'g'}})
	m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'n'}})
	if m.PromptMode != promptQuickNew {
		t.Errorf("gn should enter promptQuickNew; got %d", m.PromptMode)
	}
}

func TestStatusBandSort(t *testing.T) {
	dir := t.TempDir()
	tree, _ := store.Load(dir)
	tree.Create(tree.Root, "Todo Task")
	doing, _ := tree.Create(tree.Root, "Doing Task")
	tree.SetStatus(doing, model.StatusDoing)
	done, _ := tree.Create(tree.Root, "Done Task")
	tree.SetStatus(done, model.StatusDone)
	blocked, _ := tree.Create(tree.Root, "Blocked Task")
	tree.SetStatus(blocked, model.StatusBlocked)

	th, _ := theme.Resolve(&config.Config{ThemeName: "dracula"})
	m := newModel(tree, th, &config.Config{}).(*Model)
	m.Update(tea.WindowSizeMsg{Width: 120, Height: 40})

	visible := m.visibleSiblings()
	want := []string{"doing-task", "blocked-task", "todo-task", "done-task"}
	if len(visible) != len(want) {
		t.Fatalf("got %d siblings, want %d", len(visible), len(want))
	}
	for i, w := range want {
		if visible[i].Slug != w {
			t.Errorf("position %d: got %q, want %q", i, visible[i].Slug, w)
		}
	}
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

func TestDrillIntoEmptyContainer(t *testing.T) {
	dir := t.TempDir()
	tree, _ := store.Load(dir)
	tree.Create(tree.Root, "Project A") // top-level, no children
	th, _ := theme.Resolve(&config.Config{ThemeName: "dracula"})
	m := newModel(tree, th, &config.Config{}).(*Model)
	m.Update(tea.WindowSizeMsg{Width: 120, Height: 40})

	// Press l on the empty project — must drill in.
	m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'l'}})
	if m.Current == nil || m.Current.Slug != "project-a" {
		t.Fatalf("l on empty project must drill in; current=%v", m.Current)
	}

	// Now press n + type + Enter → child should be created INSIDE project-a.
	m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'n'}})
	for _, r := range "First Note" {
		m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{r}})
	}
	m.Update(tea.KeyMsg{Type: tea.KeyEnter})

	if len(m.Current.Children) != 1 || m.Current.Children[0].Slug != "first-note" {
		t.Errorf("expected child created under project-a, got %+v", m.Current.Children)
	}
	if l := len(tree.Root.Children); l != 1 {
		t.Errorf("root should still have 1 child, got %d", l)
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
	// gg jumps to top → that's `..` (proper two-key chord).
	m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'g'}})
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
	// Drill into "work" (a project) so the cursor lands on its task child.
	tm.Send(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'l'}})
	tm.Send(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'x'}}) // todo -> doing
	tm.Send(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'q'}})
	tm.WaitFinished(t, teatest.WithFinalTimeout(2*time.Second))

	var work *model.Node
	for _, c := range tree.Root.Children {
		if c.Slug == "work" {
			work = c
		}
	}
	if work == nil || len(work.Children) == 0 {
		t.Fatal("expected work to have children")
	}
	if work.Children[0].FM.Status != model.StatusDoing {
		t.Errorf("status = %q, want doing", work.Children[0].FM.Status)
	}
}

func TestStatusCycleNoOpOnContainer(t *testing.T) {
	tree, th, cfg := setup(t)
	m := newModel(tree, th, cfg).(*Model)
	m.Update(tea.WindowSizeMsg{Width: 120, Height: 40})
	if m.selectedNode() == nil || !m.selectedNode().IsContainer() {
		t.Fatalf("expected selection to be a container, got %v", m.selectedNode())
	}
	before := m.selectedNode().FM.Status
	m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'x'}})
	if m.selectedNode().FM.Status != before {
		t.Errorf("container status should not change, got %q want %q",
			m.selectedNode().FM.Status, before)
	}
	if m.ActionOut != nil {
		t.Errorf("expected silent no-op, got ActionOut=%+v", m.ActionOut)
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
	alpha, _ := tree.Create(tree.Root, "Alpha")
	alpha.FM.Type = model.TypeTask
	beta, _ := tree.Create(tree.Root, "Beta")
	beta.FM.Type = model.TypeTask
	tree.SetStatus(beta, model.StatusDone)
	gamma, _ := tree.Create(tree.Root, "Gamma")
	gamma.FM.Type = model.TypeTask
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
