package app

import (
	"path/filepath"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/x/exp/teatest"
	"github.com/mwdomino/cascade/internal/action"
	"github.com/mwdomino/cascade/internal/config"
	"github.com/mwdomino/cascade/internal/model"
	"github.com/mwdomino/cascade/internal/store"
	"github.com/mwdomino/cascade/internal/theme"
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
