package theme

import (
	"strings"
	"testing"

	"github.com/charmbracelet/glamour"
	"github.com/mwdomino/cascade/internal/config"
)

func TestRenderHeadingsHaveColor(t *testing.T) {
	th, _ := Resolve(&config.Config{ThemeName: "dracula"})
	body := "# Heading One\n\n## Heading Two\n\n### Heading Three\n"
	r, err := glamour.NewTermRenderer(glamour.WithStyles(th.GlamourStyle()), glamour.WithWordWrap(60))
	if err != nil {
		t.Fatal(err)
	}
	out, err := r.Render(body)
	if err != nil {
		t.Fatal(err)
	}
	// Dracula heading_h1=#bd93f9 (rgb 189;147;249), h2=#ff79c6 (255;121;198),
	// h3=#8be9fd (139;233;253). The SGR escape should embed those triplets.
	for _, want := range []string{"189;147;249", "255;121;198", "139;233;253"} {
		if !strings.Contains(out, want) {
			t.Errorf("expected color %q in rendered headings; got:\n%s", want, out)
		}
	}
	if strings.Contains(out, "# Heading One") {
		t.Errorf("literal '# ' prefix should have been stripped:\n%s", out)
	}
}

// TestRenderSmokeMatchesUserExample renders the literal markdown the user
// reported as not working and asserts the rendered output picks up the
// expected escape sequences for italic / bold / strikethrough and shows
// the list bullet.
func TestRenderSmokeMatchesUserExample(t *testing.T) {
	th, _ := Resolve(&config.Config{ThemeName: "dracula"})
	body := "intro\n\n- is\n- a\n- list\n\n~~strikethrough~~\n\n*italic*\n\n**bold**\n"
	r, err := glamour.NewTermRenderer(glamour.WithStyles(th.GlamourStyle()), glamour.WithWordWrap(60))
	if err != nil {
		t.Fatal(err)
	}
	out, err := r.Render(body)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, "•") {
		t.Errorf("expected bullet glyph in rendered list:\n%s", out)
	}
	// SGR 3 = italic, SGR 1 = bold, SGR 9 = crossed-out. These appear as
	// escape codes inside `out`.
	// glamour writes SGR escapes that combine the color and the style
	// attribute (e.g. "\x1b[38;2;…;3m" rather than a bare "\x1b[3m"), so
	// match the trailing attribute followed by the SGR terminator.
	checks := []struct {
		name, want string
	}{
		{"italic", ";3m"},
		{"bold", ";1m"},
		{"strike", ";9m"},
	}
	for _, c := range checks {
		if !strings.Contains(out, c.want) {
			t.Errorf("%s: no %q escape in:\n%s", c.name, c.want, out)
		}
	}
}
