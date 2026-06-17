package prompt

import (
	"testing"

	"github.com/mwdomino/cascade/internal/config"
	"github.com/mwdomino/cascade/internal/theme"
)

func TestFocusBlur(t *testing.T) {
	th, _ := theme.Resolve(&config.Config{ThemeName: "dracula"})
	m := New(th)
	if m.Focused() {
		t.Error("should start blurred")
	}
	m.Focus()
	if !m.Focused() {
		t.Error("should be focused")
	}
	m.Blur()
	if m.Focused() {
		t.Error("should be blurred again")
	}
}
