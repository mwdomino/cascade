package theme

import (
	"fmt"

	"github.com/charmbracelet/glamour/ansi"
	"github.com/charmbracelet/glamour/styles"
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

// GlamourStyle starts from glamour's bundled Dracula config (which already
// has sensible defaults for lists, emphasis, strikethrough, blockquotes, and
// code blocks) and overrides only the slots cascade lets users theme via
// yaml: headings (per level), code/link/list color, and the task checkbox
// glyphs/colors.
func (t *Theme) GlamourStyle() ansi.StyleConfig {
	str := func(c lipgloss.Color) *string { s := string(c); return &s }
	zero := uint(0)
	cfg := styles.DraculaStyleConfig

	// Cascade renders the title block itself (lipgloss-styled) and pads the
	// details pane, so we don't want glamour's "# "/"## " literal prefixes
	// or the outer document margin around the body.
	cfg.Document.Margin = &zero
	cfg.Document.BlockPrefix = ""
	cfg.Document.BlockSuffix = ""

	headingColor := func(level *ansi.StyleBlock, c lipgloss.Color) {
		if c == "" {
			c = t.Markdown.Heading
		}
		level.Prefix = "" // drop the literal "# "/"## " glamour prepends
		if c == "" {
			return
		}
		level.Color = str(c)
		level.Bold = boolPtr(true)
	}
	headingColor(&cfg.H1, t.Markdown.HeadingH1)
	headingColor(&cfg.H2, t.Markdown.HeadingH2)
	headingColor(&cfg.H3, t.Markdown.HeadingH3)
	headingColor(&cfg.H4, t.Markdown.HeadingH4)
	headingColor(&cfg.H5, t.Markdown.HeadingH5)
	headingColor(&cfg.H6, t.Markdown.HeadingH6)
	if t.Markdown.Heading != "" {
		cfg.Heading.Color = str(t.Markdown.Heading)
	}

	if t.Markdown.Code != "" {
		cfg.Code.Color = str(t.Markdown.Code)
		cfg.CodeBlock.Color = str(t.Markdown.Code)
	}
	if t.Markdown.Link != "" {
		cfg.Link.Color = str(t.Markdown.Link)
		cfg.LinkText.Color = str(t.Markdown.Link)
	}
	if t.Markdown.List != "" {
		cfg.Item.Color = str(t.Markdown.List)
		cfg.Enumeration.Color = str(t.Markdown.List)
	}
	if t.Palette.Fg != "" {
		cfg.Document.Color = str(t.Palette.Fg)
		// Intentionally don't override Text.Color: when a Text node lives
		// inside a styled block (heading, blockquote, …), glamour's cascade
		// gives a non-nil child Color priority over the parent. Setting
		// Text.Color globally would defeat per-block colors. Document.Color
		// is enough as the baseline for unstyled paragraph text.
	}

	// Replace task glyphs with cascade's status icons in cascade's status
	// colors. Glamour's Task slot uses a single foreground color for both
	// ticked and unticked; we bake per-state colors into the prefix strings.
	doneColor := t.Markdown.CheckboxDone
	if doneColor == "" {
		doneColor = t.Status.Done
	}
	todoColor := t.Markdown.CheckboxTodo
	if todoColor == "" {
		todoColor = t.Palette.Dim
	}
	cfg.Task.Ticked = lipgloss.NewStyle().Foreground(doneColor).Render("✓ ")
	cfg.Task.Unticked = lipgloss.NewStyle().Foreground(todoColor).Render("○ ")

	return cfg
}

func boolPtr(b bool) *bool { return &b }
