package theme

import (
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
