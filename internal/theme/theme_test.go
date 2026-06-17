package theme

import (
	"strings"
	"testing"

	"github.com/mwdomino/cascade/internal/config"
	"github.com/mwdomino/cascade/internal/model"
)

func TestResolveBuiltin(t *testing.T) {
	cfg := &config.Config{ThemeName: "dracula"}
	th, err := Resolve(cfg)
	if err != nil {
		t.Fatal(err)
	}
	if th.Name != "dracula" {
		t.Errorf("name=%q", th.Name)
	}
	if string(th.Palette.Bg) != "#282a36" {
		t.Errorf("bg=%q", th.Palette.Bg)
	}
}

func TestResolveDefaultIsDracula(t *testing.T) {
	cfg := &config.Config{}
	th, err := Resolve(cfg)
	if err != nil {
		t.Fatal(err)
	}
	if th.Name != "dracula" {
		t.Errorf("default theme = %q, want dracula", th.Name)
	}
}

func TestResolveUnknown(t *testing.T) {
	cfg := &config.Config{ThemeName: "doesnotexist"}
	if _, err := Resolve(cfg); err == nil {
		t.Error("expected error for unknown theme")
	}
}

func TestGlamourStyleHeadingsAndTasks(t *testing.T) {
	th, _ := Resolve(&config.Config{ThemeName: "dracula"})
	s := th.GlamourStyle()
	if s.H1.Color == nil || *s.H1.Color != "#bd93f9" {
		t.Errorf("H1 color: %v", s.H1.Color)
	}
	if s.H2.Color == nil || *s.H2.Color != "#ff79c6" {
		t.Errorf("H2 color: %v", s.H2.Color)
	}
	if s.H3.Color == nil || *s.H3.Color != "#8be9fd" {
		t.Errorf("H3 color: %v", s.H3.Color)
	}
	if !strings.Contains(s.Task.Ticked, "✓") {
		t.Errorf("Task.Ticked missing ✓: %q", s.Task.Ticked)
	}
	if !strings.Contains(s.Task.Unticked, "○") {
		t.Errorf("Task.Unticked missing ○: %q", s.Task.Unticked)
	}
}

func TestStatusGlyphAllStatuses(t *testing.T) {
	th, _ := Resolve(&config.Config{ThemeName: "dracula"})
	for _, s := range []model.Status{
		model.StatusTodo, model.StatusDoing, model.StatusDone, model.StatusBlocked,
	} {
		if g := th.StatusGlyph(s); g == "" {
			t.Errorf("no glyph for %q", s)
		}
	}
}
