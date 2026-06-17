package theme

import (
	"fmt"

	"github.com/charmbracelet/glamour/ansi"
	"github.com/charmbracelet/lipgloss"
	"github.com/mwdomino/cascade/internal/config"
	"github.com/mwdomino/cascade/internal/model"
	"gopkg.in/yaml.v3"
)

type Theme struct {
	Name      string
	Palette   Palette
	Status    StatusColors
	Selection SelectionColors
	Markdown  MarkdownColors
}

type Palette struct {
	Bg     lipgloss.Color `yaml:"bg"`
	Fg     lipgloss.Color `yaml:"fg"`
	Dim    lipgloss.Color `yaml:"dim"`
	Border lipgloss.Color `yaml:"border"`
	Accent lipgloss.Color `yaml:"accent"`
}

type StatusColors struct {
	Todo    lipgloss.Color `yaml:"todo"`
	Doing   lipgloss.Color `yaml:"doing"`
	Done    lipgloss.Color `yaml:"done"`
	Blocked lipgloss.Color `yaml:"blocked"`
}

type SelectionColors struct {
	CursorBg    lipgloss.Color `yaml:"cursor_bg"`
	SearchMatch lipgloss.Color `yaml:"search_match"`
}

type MarkdownColors struct {
	Heading lipgloss.Color `yaml:"heading"`
	Code    lipgloss.Color `yaml:"code"`
	Link    lipgloss.Color `yaml:"link"`
	List    lipgloss.Color `yaml:"list"`
}

func Resolve(cfg *config.Config) (*Theme, error) {
	if cfg.ThemeInline != nil {
		return decodeFromMap(cfg.ThemeInline)
	}
	name := cfg.ThemeName
	if name == "" {
		name = "dracula"
	}
	data, err := builtinFS.ReadFile(name + ".yaml")
	if err != nil {
		return nil, fmt.Errorf("unknown theme %q", name)
	}
	var th Theme
	if err := yaml.Unmarshal(data, &th); err != nil {
		return nil, err
	}
	if th.Name == "" {
		th.Name = name
	}
	return &th, nil
}

func decodeFromMap(m map[string]any) (*Theme, error) {
	raw, err := yaml.Marshal(m)
	if err != nil {
		return nil, err
	}
	var th Theme
	if err := yaml.Unmarshal(raw, &th); err != nil {
		return nil, err
	}
	if th.Name == "" {
		th.Name = "inline"
	}
	return &th, nil
}

func (t *Theme) StatusGlyph(s model.Status) string {
	var (
		color lipgloss.Color
		ch    string
	)
	switch s {
	case model.StatusTodo:
		color, ch = t.Status.Todo, "○"
	case model.StatusDoing:
		color, ch = t.Status.Doing, "◐"
	case model.StatusDone:
		color, ch = t.Status.Done, "✓"
	case model.StatusBlocked:
		color, ch = t.Status.Blocked, "✗"
	default:
		color, ch = t.Palette.Dim, "·"
	}
	return lipgloss.NewStyle().Foreground(color).Render(ch)
}

func (t *Theme) GlamourStyle() ansi.StyleConfig {
	str := func(c lipgloss.Color) *string { s := string(c); return &s }
	bp := uint(0)
	return ansi.StyleConfig{
		Document: ansi.StyleBlock{
			StylePrimitive: ansi.StylePrimitive{Color: str(t.Palette.Fg)},
			Margin:         &bp,
		},
		Heading: ansi.StyleBlock{
			StylePrimitive: ansi.StylePrimitive{Color: str(t.Markdown.Heading), Bold: boolPtr(true)},
		},
		Code: ansi.StyleBlock{
			StylePrimitive: ansi.StylePrimitive{Color: str(t.Markdown.Code)},
		},
		Link: ansi.StylePrimitive{Color: str(t.Markdown.Link), Underline: boolPtr(true)},
		Item: ansi.StylePrimitive{Color: str(t.Markdown.List)},
	}
}

func boolPtr(b bool) *bool { return &b }
