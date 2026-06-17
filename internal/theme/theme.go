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
	Heading      lipgloss.Color `yaml:"heading"`
	HeadingH1    lipgloss.Color `yaml:"heading_h1"`
	HeadingH2    lipgloss.Color `yaml:"heading_h2"`
	HeadingH3    lipgloss.Color `yaml:"heading_h3"`
	HeadingH4    lipgloss.Color `yaml:"heading_h4"`
	HeadingH5    lipgloss.Color `yaml:"heading_h5"`
	HeadingH6    lipgloss.Color `yaml:"heading_h6"`
	Code         lipgloss.Color `yaml:"code"`
	Link         lipgloss.Color `yaml:"link"`
	List         lipgloss.Color `yaml:"list"`
	CheckboxDone lipgloss.Color `yaml:"checkbox_done"`
	CheckboxTodo lipgloss.Color `yaml:"checkbox_todo"`
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

// NodeGlyph picks the right glyph for a node:
//   - container with every descendant task done → ✓ in done color (rollup),
//   - project → filled square in accent,
//   - folder → triangle in dim,
//   - task → status icon.
func (t *Theme) NodeGlyph(n *model.Node) string {
	if n.IsContainer() && n.EffectivelyDone() {
		return lipgloss.NewStyle().Foreground(t.Status.Done).Render("✓")
	}
	switch n.EffectiveType() {
	case model.TypeProject:
		return lipgloss.NewStyle().Foreground(t.Palette.Accent).Bold(true).Render("■")
	case model.TypeFolder:
		return lipgloss.NewStyle().Foreground(t.Palette.Dim).Render("▸")
	default:
		return t.StatusGlyph(n.FM.Status)
	}
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

	headingBlock := func(c lipgloss.Color) ansi.StyleBlock {
		color := c
		if color == "" {
			color = t.Markdown.Heading
		}
		return ansi.StyleBlock{
			StylePrimitive: ansi.StylePrimitive{Color: str(color), Bold: boolPtr(true)},
		}
	}

	doneColor := t.Markdown.CheckboxDone
	if doneColor == "" {
		doneColor = t.Status.Done
	}
	todoColor := t.Markdown.CheckboxTodo
	if todoColor == "" {
		todoColor = t.Palette.Dim
	}

	return ansi.StyleConfig{
		Document: ansi.StyleBlock{
			StylePrimitive: ansi.StylePrimitive{Color: str(t.Palette.Fg)},
			Margin:         &bp,
		},
		Heading: ansi.StyleBlock{
			StylePrimitive: ansi.StylePrimitive{Color: str(t.Markdown.Heading), Bold: boolPtr(true)},
		},
		H1: headingBlock(t.Markdown.HeadingH1),
		H2: headingBlock(t.Markdown.HeadingH2),
		H3: headingBlock(t.Markdown.HeadingH3),
		H4: headingBlock(t.Markdown.HeadingH4),
		H5: headingBlock(t.Markdown.HeadingH5),
		H6: headingBlock(t.Markdown.HeadingH6),
		Code: ansi.StyleBlock{
			StylePrimitive: ansi.StylePrimitive{Color: str(t.Markdown.Code)},
		},
		Link: ansi.StylePrimitive{Color: str(t.Markdown.Link), Underline: boolPtr(true)},
		Item: ansi.StylePrimitive{Color: str(t.Markdown.List)},
		Task: ansi.StyleTask{
			StylePrimitive: ansi.StylePrimitive{Color: str(t.Markdown.List)},
			Ticked:         lipgloss.NewStyle().Foreground(doneColor).Render("✓ "),
			Unticked:       lipgloss.NewStyle().Foreground(todoColor).Render("○ "),
		},
	}
}

func boolPtr(b bool) *bool { return &b }
