package app

import (
	"fmt"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/mwdomino/cascade/internal/action"
	"github.com/mwdomino/cascade/internal/config"
	"github.com/mwdomino/cascade/internal/editor"
	"github.com/mwdomino/cascade/internal/model"
	"github.com/mwdomino/cascade/internal/store"
	"github.com/mwdomino/cascade/internal/theme"
	"github.com/mwdomino/cascade/internal/tui/breadcrumb"
	"github.com/mwdomino/cascade/internal/tui/details"
	"github.com/mwdomino/cascade/internal/tui/keys"
	"github.com/mwdomino/cascade/internal/tui/palette"
	"github.com/mwdomino/cascade/internal/tui/prompt"
	"github.com/mwdomino/cascade/internal/tui/search"
	"github.com/mwdomino/cascade/internal/tui/sidebar"
)

type promptMode int

const (
	promptNone        promptMode = iota
	promptNew
	promptQuickNew
	promptRename
	promptSearchLocal
	promptSearchGlobal
)

type editorClosedMsg struct{}

type actionDoneMsg struct {
	Name   string
	Result action.Result
}

// runActionCmd executes the action in a goroutine and emits actionDoneMsg
// when complete so the TUI stays responsive while shell commands run.
func runActionCmd(act action.Action, n *model.Node) tea.Cmd {
	return func() tea.Msg {
		res, _ := act.Run(n)
		return actionDoneMsg{Name: act.Name, Result: res}
	}
}

type Model struct {
	Tree    *store.Tree
	Theme   *theme.Theme
	Cfg     *config.Config
	Keys    keys.Map
	Current *model.Node // parent of the displayed siblings
	Cursor  int
	ShowDone bool

	Sidebar    sidebar.Model
	Details    details.Model
	Breadcrumb breadcrumb.Model

	PromptMode promptMode
	Prompt     prompt.Model

	// Search state
	LocalQuery    string
	GlobalMatches []*model.Node
	GlobalMode    bool

	// Confirm overlay state
	ConfirmMode bool
	ConfirmHard bool
	PendingDD   bool
	DDDeadline  time.Time

	// g-chord state (gg = Top, gn = QuickNew)
	PendingG  bool
	GDeadline time.Time

	// Palette + action state
	ActionReg   *action.Registry
	PaletteMode bool
	Palette     palette.Model
	ActionOut       *action.Result
	ActionByKey     map[string]action.Action
	ActionRunning   string // name of the action currently executing, "" when idle

	HelpMode bool

	// Move-to picker (centered overlay reusing palette.Model)
	MovePickerMode bool
	MovePicker     palette.Model

	// Checkbox toggle overlay state
	ToggleMode bool

	Width, Height int
}

func New(tree *store.Tree, th *theme.Theme, cfg *config.Config, reg *action.Registry) tea.Model {
	byKey := make(map[string]action.Action)
	for name, def := range reg.Defs() {
		if def.Keybind != "" {
			byKey[def.Keybind] = action.Action{Name: name, Def: def}
		}
	}
	return &Model{
		Tree:        tree,
		Theme:       th,
		Cfg:         cfg,
		Keys:        keys.Default(),
		Current:     tree.Root,
		ShowDone:    true, // visible by default; Z hides them
		Sidebar:     sidebar.Model{Theme: th},
		Details:     details.Model{Theme: th},
		Breadcrumb:  breadcrumb.Model{Theme: th},
		Prompt:      prompt.New(th),
		ActionReg:   reg,
		ActionByKey: byKey,
	}
}

func (m *Model) Init() tea.Cmd { return nil }

func (m *Model) visibleSiblings() []*model.Node {
	if m.GlobalMode {
		return m.GlobalMatches
	}
	sibs := m.Current.Children
	if m.LocalQuery != "" {
		sibs = search.LocalFilter(sibs, m.LocalQuery)
	}
	if !m.ShowDone {
		filtered := make([]*model.Node, 0, len(sibs))
		for _, n := range sibs {
			// Hide anything that's effectively done: done leaf tasks AND
			// fully-rolled-up containers.
			if n.EffectivelyDone() {
				continue
			}
			filtered = append(filtered, n)
		}
		sibs = filtered
	}
	// Stable-sort by status priority so active work bubbles up: doing first,
	// then blocked, then todo, then effectively-done at the bottom. Manual
	// prefix order is preserved within each band by the stable sort.
	if len(sibs) > 1 {
		sorted := make([]*model.Node, len(sibs))
		copy(sorted, sibs)
		sort.SliceStable(sorted, func(i, j int) bool {
			return statusBand(sorted[i]) < statusBand(sorted[j])
		})
		sibs = sorted
	}
	return sibs
}

// statusBand returns a sort key for the visible-tier ordering:
//
//	0 = doing       — active work, top
//	1 = blocked     — needs attention
//	2 = todo (or container that isn't fully done)
//	3 = effectively done — bottom
func statusBand(n *model.Node) int {
	if n.EffectivelyDone() {
		return 3
	}
	if n.EffectiveType() == model.TypeTask {
		switch n.FM.Status {
		case model.StatusDoing:
			return 0
		case model.StatusBlocked:
			return 1
		}
	}
	return 2
}

func (m *Model) selectedNode() *model.Node {
	sibs := m.visibleSiblings()
	idx := m.childIndex()
	if idx < 0 || idx >= len(sibs) {
		return nil
	}
	return sibs[idx]
}

// hasDotDot reports whether a `..` row should appear at the top of the sidebar.
// Suppressed in global-search results and when a local filter is active so the
// match list isn't interrupted.
func (m *Model) hasDotDot() bool {
	if m.GlobalMode || m.LocalQuery != "" {
		return false
	}
	return m.Current != nil && m.Current.Parent != nil
}

func (m *Model) totalRows() int {
	n := len(m.visibleSiblings())
	if m.hasDotDot() {
		n++
	}
	return n
}

func (m *Model) cursorMax() int {
	n := m.totalRows()
	if n <= 0 {
		return 0
	}
	return n - 1
}

func (m *Model) cursorIsDotDot() bool {
	return m.hasDotDot() && m.Cursor == 0
}

// childIndex maps the unified Cursor to an index into visibleSiblings(),
// or -1 when the cursor is on the synthetic `..` row.
func (m *Model) childIndex() int {
	if m.cursorIsDotDot() {
		return -1
	}
	if m.hasDotDot() {
		return m.Cursor - 1
	}
	return m.Cursor
}

// cursorAtChild returns the cursor position that points at `target` within
// m.Current.Children, accounting for an optional `..` row.
func (m *Model) cursorAtChild(target *model.Node) int {
	for i, c := range m.Current.Children {
		if c == target {
			if m.hasDotDot() {
				return i + 1
			}
			return i
		}
	}
	if m.hasDotDot() {
		return 1
	}
	return 0
}

// initialDrillCursor returns the starting cursor for a freshly-entered tier:
// the first real child (skipping `..` if present), or 0 if empty.
func (m *Model) initialDrillCursor() int {
	if m.hasDotDot() && len(m.visibleSiblings()) > 0 {
		return 1
	}
	return 0
}

// goUp moves to the parent tier and positions the cursor on the node we came from.
func (m *Model) goUp() {
	if m.Current == nil || m.Current.Parent == nil {
		return
	}
	prev := m.Current
	m.Current = m.Current.Parent
	m.Cursor = m.cursorAtChild(prev)
}

// restoreNav attempts to restore Current and Cursor to the saved paths after a
// tree reload. Falls back to root/0 if the paths no longer exist.
func (m *Model) restoreNav(savedCurrent, savedSelected string) {
	restored := m.Tree.NodeAt(savedCurrent)
	if restored == nil {
		m.Current = m.Tree.Root
		m.Cursor = 0
		return
	}
	m.Current = restored
	if savedSelected != "" {
		sel := m.Tree.NodeAt(savedSelected)
		if sel != nil {
			for i, c := range m.Current.Children {
				if c == sel {
					m.Cursor = i
					return
				}
			}
		}
	}
	m.Cursor = 0
}

func (m *Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case editorClosedMsg:
		savedCurrent := m.Current.Path
		savedSelected := ""
		if sel := m.selectedNode(); sel != nil {
			savedSelected = sel.Path
		}
		_ = m.Tree.Reload()
		m.restoreNav(savedCurrent, savedSelected)
		return m, nil
	case actionDoneMsg:
		res := msg.Result
		m.ActionOut = &res
		m.ActionRunning = ""
		return m, nil
	case tea.WindowSizeMsg:
		m.Width = msg.Width
		m.Height = msg.Height
		sw := msg.Width / 5
		if sw < 20 {
			sw = 20
		}
		m.Sidebar.Width = sw
		m.Sidebar.Height = msg.Height - 2
		m.Details.Width = msg.Width - sw - 2
		m.Details.Height = msg.Height - 2
	case tea.KeyMsg:
		// Clear transient action output on any keypress.
		if m.ActionOut != nil {
			m.ActionOut = nil
			return m, nil
		}

		// Help overlay: any key closes it.
		if m.HelpMode {
			m.HelpMode = false
			return m, nil
		}

		// Checkbox toggle overlay: digit toggles the labeled checkbox; any
		// other key (incl. Esc) closes the overlay.
		if m.ToggleMode {
			if d, ok := digitPressed(msg.String()); ok {
				m.handleCheckboxToggle(d)
			}
			m.ToggleMode = false
			m.Details.LabelCheckboxes = false
			return m, nil
		}

		// Palette mode: forward keys to palette, handle Esc.
		if m.PaletteMode {
			if msg.String() == "esc" {
				m.PaletteMode = false
				return m, nil
			}
			var cmd tea.Cmd
			m.Palette, cmd = m.Palette.Update(msg)
			return m, cmd
		}

		// Move-to picker mode: forward keys to the picker, handle Esc.
		if m.MovePickerMode {
			if msg.String() == "esc" {
				m.MovePickerMode = false
				return m, nil
			}
			var cmd tea.Cmd
			m.MovePicker, cmd = m.MovePicker.Update(msg)
			return m, cmd
		}

		// Confirm overlay: handle before everything else.
		if m.ConfirmMode {
			switch msg.String() {
			case "y", "Y", "enter":
				n := m.selectedNode()
				if n != nil {
					var err error
					if m.ConfirmHard {
						err = m.Tree.HardDelete(n)
					} else {
						err = m.Tree.SoftDelete(n)
					}
					if err == nil && m.Cursor > m.cursorMax() {
						m.Cursor = m.cursorMax()
					}
				}
				m.ConfirmMode = false
			case "n", "N", "esc":
				m.ConfirmMode = false
			}
			return m, nil
		}

		// When prompt is active, forward all keys to the prompt except Enter/Esc.
		if m.PromptMode != promptNone {
			switch msg.String() {
			case "enter":
				return m, m.submitPrompt()
			case "esc":
				m.PromptMode = promptNone
				m.Prompt.Blur()
				return m, nil
			}
			var cmd tea.Cmd
			m.Prompt, cmd = m.Prompt.Update(msg)
			return m, cmd
		}

		// Esc outside prompt clears active search state.
		if msg.String() == "esc" && (m.LocalQuery != "" || m.GlobalMode) {
			m.LocalQuery = ""
			m.GlobalMode = false
			m.GlobalMatches = nil
			m.Cursor = 0
			return m, nil
		}

		// g-chord double-tap (gg = Top, gn = QuickNew)
		{
			chordPending := m.PendingG && time.Now().Before(m.GDeadline)
			if msg.String() == "g" {
				if chordPending {
					m.PendingG = false
					m.Cursor = 0 // Top
					return m, nil
				}
				m.PendingG = true
				m.GDeadline = time.Now().Add(500 * time.Millisecond)
				return m, nil
			}
			if chordPending && msg.String() == "n" {
				m.PendingG = false
				if m.PromptMode == promptNone {
					m.PromptMode = promptQuickNew
					m.Prompt.SetLabel("inbox:")
					m.Prompt.Reset()
					m.Prompt.Focus()
					return m, nil
				}
			}
			// Any other key (or expired chord): cancel pending state and fall through.
			m.PendingG = false
		}

		// dd double-tap (raw key check before the switch)
		if msg.String() == "d" {
			if m.PendingDD && time.Now().Before(m.DDDeadline) {
				m.PendingDD = false
				m.ConfirmMode = true
				m.ConfirmHard = false
				return m, nil
			}
			m.PendingDD = true
			m.DDDeadline = time.Now().Add(500 * time.Millisecond)
			return m, nil
		}
		// Any non-d key resets the pending dd state.
		m.PendingDD = false

		// Direct keybind dispatch (palette closed, prompt idle).
		if m.PromptMode == promptNone {
			if act, ok := m.ActionByKey[msg.String()]; ok {
				if sel := m.selectedNode(); sel != nil {
					m.ActionRunning = act.Name
					return m, runActionCmd(act, sel)
				}
			}
		}

		switch {
		case key.Matches(msg, m.Keys.Palette):
			if m.PromptMode == promptNone {
				m.PaletteMode = true
				m.Palette = palette.New(m.Theme)
				m.Palette.SetItems(m.paletteItems())
				return m, nil
			}
		case key.Matches(msg, m.Keys.Quit):
			return m, tea.Quit
		case key.Matches(msg, m.Keys.Up):
			if m.Cursor > 0 {
				m.Cursor--
			}
		case key.Matches(msg, m.Keys.Down):
			if m.Cursor < m.cursorMax() {
				m.Cursor++
			}
		case key.Matches(msg, m.Keys.In):
			if m.GlobalMode {
				if len(m.GlobalMatches) > 0 && m.Cursor < len(m.GlobalMatches) {
					target := m.GlobalMatches[m.Cursor]
					m.Current = target.Parent
					m.Cursor = m.cursorAtChild(target)
					m.GlobalMode = false
					m.GlobalMatches = nil
				}
				return m, nil
			}
			if m.cursorIsDotDot() {
				m.goUp()
				return m, nil
			}
			// Drill into anything (even empty containers and leaves) so the user
			// can always add children. A leaf turns into a folder once it has one.
			if n := m.selectedNode(); n != nil {
				m.Current = n
				m.Cursor = m.initialDrillCursor()
			}
		case key.Matches(msg, m.Keys.Out):
			m.goUp()
		case key.Matches(msg, m.Keys.Top):
			m.Cursor = 0
		case key.Matches(msg, m.Keys.Bottom):
			m.Cursor = m.cursorMax()
		case key.Matches(msg, m.Keys.Refresh):
			savedCurrent := m.Current.Path
			savedSelected := ""
			if sel := m.selectedNode(); sel != nil {
				savedSelected = sel.Path
			}
			if err := m.Tree.Reload(); err == nil {
				m.restoreNav(savedCurrent, savedSelected)
			}
		case key.Matches(msg, m.Keys.ToggleDone):
			m.ShowDone = !m.ShowDone
		case key.Matches(msg, m.Keys.StatusCycle):
			if n := m.selectedNode(); n != nil && !n.IsContainer() {
				if err := m.Tree.SetStatus(n, n.FM.Status.Cycle()); err != nil {
					m.ActionOut = &action.Result{Stderr: err.Error(), ExitCode: 1}
				}
			}
			// On containers x is a silent no-op: their status is rolled up
			// from descendants automatically.
		case key.Matches(msg, m.Keys.MoveUp):
			if n := m.selectedNode(); n != nil {
				if err := m.Tree.MoveUp(n); err == nil && m.childIndex() > 0 {
					m.Cursor--
				}
			}
		case key.Matches(msg, m.Keys.MoveDown):
			if n := m.selectedNode(); n != nil {
				if err := m.Tree.MoveDown(n); err == nil && m.childIndex() < len(m.Current.Children)-1 {
					m.Cursor++
				}
			}
		case key.Matches(msg, m.Keys.HardDelete):
			if m.selectedNode() != nil {
				m.ConfirmMode = true
				m.ConfirmHard = true
			}
		case key.Matches(msg, m.Keys.New):
			if m.PromptMode == promptNone {
				m.PromptMode = promptNew
				m.Prompt.SetLabel("new:")
				m.Prompt.Reset()
				m.Prompt.Focus()
				return m, nil
			}
		case key.Matches(msg, m.Keys.QuickNew):
			if m.PromptMode == promptNone {
				m.PromptMode = promptQuickNew
				m.Prompt.SetLabel("inbox:")
				m.Prompt.Reset()
				m.Prompt.Focus()
				return m, nil
			}
		case key.Matches(msg, m.Keys.Rename):
			if m.PromptMode == promptNone && m.selectedNode() != nil {
				m.PromptMode = promptRename
				m.Prompt.SetLabel("rename:")
				m.Prompt.Reset()
				m.Prompt.SetValue(m.selectedNode().Title())
				m.Prompt.Focus()
				return m, nil
			}
		case key.Matches(msg, m.Keys.MoveTo):
			if m.PromptMode == promptNone && !m.MovePickerMode && m.selectedNode() != nil {
				m.openMovePicker()
				return m, nil
			}
		case key.Matches(msg, m.Keys.Edit):
			if n := m.selectedNode(); n != nil {
				target := filepath.Join(n.Path, "index.md")
				return m, tea.ExecProcess(externalEditorCmd(target), func(err error) tea.Msg {
					return editorClosedMsg{}
				})
			}
		case key.Matches(msg, m.Keys.SearchLocal):
			if m.PromptMode == promptNone {
				m.PromptMode = promptSearchLocal
				m.Prompt.SetLabel("/")
				m.Prompt.Reset()
				m.Prompt.Focus()
				return m, nil
			}
		case key.Matches(msg, m.Keys.SearchGlobal):
			if m.PromptMode == promptNone {
				m.PromptMode = promptSearchGlobal
				m.Prompt.SetLabel("find:")
				m.Prompt.Reset()
				m.Prompt.Focus()
				return m, nil
			}
		case key.Matches(msg, m.Keys.Help):
			m.HelpMode = true
			return m, nil
		case key.Matches(msg, m.Keys.ToggleCheckbox):
			if n := m.selectedNode(); n != nil && details.CountCheckboxes(n.Body) > 0 {
				m.ToggleMode = true
				m.Details.LabelCheckboxes = true
			}
			return m, nil
		case key.Matches(msg, m.Keys.ScrollDown):
			step := m.Details.Height / 2
			if step < 1 {
				step = 1
			}
			m.Details.ScrollDown(step)
		case key.Matches(msg, m.Keys.ScrollUp):
			step := m.Details.Height / 2
			if step < 1 {
				step = 1
			}
			m.Details.ScrollUp(step)
		}
	}
	return m, nil
}

func (m *Model) submitPrompt() tea.Cmd {
	val := strings.TrimSpace(m.Prompt.Value())
	switch m.PromptMode {
	case promptNew:
		if val != "" {
			if _, err := m.Tree.Create(m.Current, val); err == nil {
				m.Cursor = len(m.Current.Children) - 1
			} else {
				m.ActionOut = &action.Result{Stderr: err.Error(), ExitCode: 1}
			}
		}
	case promptQuickNew:
		if val != "" {
			target := m.inboxNode()
			if target != nil {
				if _, err := m.Tree.Create(target, val); err != nil {
					m.ActionOut = &action.Result{Stderr: err.Error(), ExitCode: 1}
				}
			}
		}
	case promptRename:
		if val != "" && m.selectedNode() != nil {
			if err := m.Tree.Rename(m.selectedNode(), val); err != nil {
				m.ActionOut = &action.Result{Stderr: err.Error(), ExitCode: 1}
			}
		}
	case promptSearchLocal:
		m.LocalQuery = val
		m.Cursor = 0
	case promptSearchGlobal:
		m.GlobalMatches = search.GlobalFuzzy(m.Tree.AllNodes(), val)
		m.GlobalMode = true
		m.Cursor = 0
	}
	m.PromptMode = promptNone
	m.Prompt.Blur()
	return nil
}

func (m *Model) inboxNode() *model.Node {
	inboxName := strings.TrimSpace(m.Cfg.Inbox)
	if inboxName == "" {
		inboxName = "999-inbox"
	}
	// Strip the numeric prefix if present so we compare by slug.
	_, slug, ok := store.ParsePrefix(inboxName)
	if !ok {
		slug = inboxName // user gave a bare slug like "inbox"
	}
	for _, c := range m.Tree.Root.Children {
		if c.Slug == slug {
			return c
		}
	}
	// Not found — create a top-level inbox category. Use the slug as title;
	// its on-disk prefix will be assigned by Tree.Create (next gap-of-10).
	n, err := m.Tree.Create(m.Tree.Root, slug)
	if err != nil {
		return nil
	}
	return n
}

func (m *Model) paletteItems() []palette.Item {
	items := []palette.Item{
		{Name: "refresh", Run: func() tea.Cmd {
			_ = m.Tree.Reload()
			m.PaletteMode = false
			return nil
		}},
	}
	sel := m.selectedNode()
	if sel != nil {
		for _, a := range m.ActionReg.Applicable(sel) {
			act := a
			items = append(items, palette.Item{
				Name: act.Name,
				Hint: act.Def.Cmd,
				Run: func() tea.Cmd {
					m.ActionRunning = act.Name
					m.PaletteMode = false
					return runActionCmd(act, sel)
				},
			})
		}
	}
	return items
}

func externalEditorCmd(path string) *exec.Cmd {
	line := editor.EditorCmd()
	parts := strings.Fields(line)
	args := append(parts[1:], path)
	return exec.Command(parts[0], args...)
}

func (m *Model) View() string {
	if m.HelpMode {
		overlay := m.helpOverlay()
		if m.Width > 0 && m.Height > 0 {
			return lipgloss.Place(m.Width, m.Height, lipgloss.Center, lipgloss.Center, overlay)
		}
		return overlay
	}

	head := m.Breadcrumb.View(m.Current)
	hint := m.hintBar()

	// Build bottom slot (everything except palette, which renders as a
	// centered modal over the pane instead of growing the layout).
	var bottom string
	switch {
	case m.ConfirmMode:
		confirmMsg := "soft-delete?"
		if m.ConfirmHard {
			confirmMsg = "HARD DELETE? (cannot be undone)"
		}
		bottom = lipgloss.NewStyle().
			Foreground(m.Theme.Status.Blocked).
			Bold(true).
			Render(confirmMsg + " [y/N]")
	case m.PromptMode != promptNone:
		bottom = m.Prompt.View()
	case m.ActionRunning != "":
		bottom = lipgloss.NewStyle().
			Foreground(m.Theme.Status.Doing).
			Italic(true).
			Render("running " + m.ActionRunning + "…")
	case m.ActionOut != nil:
		out := fmt.Sprintf("exit=%d\n%s\n%s",
			m.ActionOut.ExitCode, m.ActionOut.Stdout, m.ActionOut.Stderr)
		bottom = lipgloss.NewStyle().Foreground(m.Theme.Palette.Dim).Render(out)
	}

	// Compute available pane height so head + pane + bottom + hint == m.Height.
	paneH := m.Height - lipgloss.Height(head) - lipgloss.Height(hint)
	if bottom != "" {
		paneH -= lipgloss.Height(bottom)
	}
	if m.Height <= 0 || paneH < 3 {
		paneH = 3
	}

	rawSide := m.Sidebar.View(m.visibleSiblings(), m.Cursor, m.ShowDone, m.hasDotDot())
	det := m.Details.View(m.selectedNode())
	det = clipLines(det, paneH)

	side := lipgloss.NewStyle().
		Width(m.Sidebar.Width).
		Height(paneH).
		BorderStyle(lipgloss.NormalBorder()).
		BorderRight(true).
		BorderForeground(m.Theme.Palette.Border).
		Render(rawSide)
	detPadded := lipgloss.NewStyle().PaddingLeft(1).Render(det)
	pane := lipgloss.JoinHorizontal(lipgloss.Top, side, detPadded)

	// Palette and move-picker render as centered modals over the pane area,
	// replacing the pane content but leaving head + hint visible.
	if m.PaletteMode {
		pane = lipgloss.Place(m.Width, paneH, lipgloss.Center, lipgloss.Center,
			m.paletteCard())
	} else if m.MovePickerMode {
		pane = lipgloss.Place(m.Width, paneH, lipgloss.Center, lipgloss.Center,
			m.movePickerCard())
	}

	parts := []string{head, pane}
	if bottom != "" {
		parts = append(parts, bottom)
	}
	if hint != "" {
		parts = append(parts, hint)
	}
	return strings.Join(parts, "\n")
}

// digitPressed parses a single-digit keypress ("1".."9").
func digitPressed(s string) (int, bool) {
	if len(s) != 1 {
		return 0, false
	}
	c := s[0]
	if c < '1' || c > '9' {
		return 0, false
	}
	return int(c - '0'), true
}

// handleCheckboxToggle flips checkbox `label` (1-9) in the selected node's
// body and writes the result back to disk. Errors surface via ActionOut.
func (m *Model) handleCheckboxToggle(label int) {
	n := m.selectedNode()
	if n == nil {
		return
	}
	newBody, ok := details.ToggleCheckbox(n.Body, label)
	if !ok {
		return
	}
	indexPath := filepath.Join(n.Path, "index.md")
	if err := store.WriteIndex(indexPath, n.FM, newBody); err != nil {
		m.ActionOut = &action.Result{Stderr: err.Error(), ExitCode: 1}
		return
	}
	n.Body = newBody
}

// openMovePicker populates and activates a centered fuzzy picker of all
// possible move targets (every node except self and descendants, plus the
// root as "(top level)").
func (m *Model) openMovePicker() {
	sel := m.selectedNode()
	if sel == nil {
		return
	}
	m.MovePickerMode = true
	m.MovePicker = palette.New(m.Theme)

	items := []palette.Item{}
	// Root is a valid target.
	items = append(items, palette.Item{
		Name: "(top level)",
		Hint: "",
		Run: func() tea.Cmd {
			if err := m.Tree.MoveTo(sel, m.Tree.Root); err != nil {
				m.ActionOut = &action.Result{Stderr: err.Error(), ExitCode: 1}
			}
			m.MovePickerMode = false
			return nil
		},
	})

	for _, t := range m.Tree.AllNodes() {
		if t == sel || isDescendant(t, sel) {
			continue
		}
		target := t // capture
		items = append(items, palette.Item{
			Name: target.Title(),
			Hint: breadcrumbPath(target),
			Run: func() tea.Cmd {
				if err := m.Tree.MoveTo(sel, target); err != nil {
					m.ActionOut = &action.Result{Stderr: err.Error(), ExitCode: 1}
				}
				m.MovePickerMode = false
				return nil
			},
		})
	}
	m.MovePicker.SetItems(items)
}

// isDescendant reports whether maybe is a descendant of root.
func isDescendant(maybe, root *model.Node) bool {
	for p := maybe.Parent; p != nil; p = p.Parent {
		if p == root {
			return true
		}
	}
	return false
}

// breadcrumbPath returns "ancestor › parent" for use as the picker hint.
func breadcrumbPath(n *model.Node) string {
	var parts []string
	for p := n.Parent; p != nil && !p.IsRoot(); p = p.Parent {
		parts = append([]string{p.Title()}, parts...)
	}
	if len(parts) == 0 {
		return ""
	}
	return strings.Join(parts, " › ")
}

// movePickerCard wraps the picker view in a rounded card like the palette.
func (m *Model) movePickerCard() string {
	header := lipgloss.NewStyle().
		Foreground(m.Theme.Palette.Accent).
		Bold(true).
		Render("move to:")
	return lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(m.Theme.Palette.Border).
		Padding(0, 2).
		Render(header + "\n" + m.MovePicker.View())
}

// paletteCard wraps the palette view in a rounded border so it reads as a
// floating modal rather than a bare list.
func (m *Model) paletteCard() string {
	return lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(m.Theme.Palette.Border).
		Padding(0, 2).
		Render(m.Palette.View())
}

// clipLines truncates s to at most n lines.
func clipLines(s string, n int) string {
	if n <= 0 {
		return ""
	}
	lines := strings.Split(s, "\n")
	if len(lines) <= n {
		return s
	}
	return strings.Join(lines[:n], "\n")
}

func (m *Model) hintBar() string {
	th := m.Theme
	sep := lipgloss.NewStyle().Foreground(th.Palette.Dim).Render(" · ")
	keyStyle := lipgloss.NewStyle().Foreground(th.Palette.Accent).Bold(true)
	descStyle := lipgloss.NewStyle().Foreground(th.Palette.Dim)
	item := func(k, d string) string { return keyStyle.Render(k) + " " + descStyle.Render(d) }

	var items []string
	switch {
	case m.ConfirmMode:
		items = []string{item("y", "confirm"), item("n/esc", "cancel")}
	case m.PaletteMode:
		items = []string{item("↑↓", "navigate"), item("enter", "run"), item("esc", "close")}
	case m.MovePickerMode:
		items = []string{item("type", "filter"), item("↑↓", "navigate"), item("enter", "move"), item("esc", "cancel")}
	case m.ToggleMode:
		items = []string{item("1-9", "toggle checkbox"), item("esc", "cancel")}
	case m.PromptMode != promptNone:
		items = []string{item("enter", "accept"), item("esc", "cancel")}
	case m.GlobalMode:
		items = []string{item("j/k", "navigate"), item("enter", "jump"), item("esc", "clear")}
	case m.LocalQuery != "":
		items = []string{item("esc", "clear filter"), item("/", "edit query")}
	default:
		items = []string{
			item("l", "drill in"),
			item("n", "new"),
			item("e", "edit"),
			item("/", "search"),
			item(":", "actions"),
			item("?", "help"),
			item("q", "quit"),
		}
	}
	return strings.Join(items, sep)
}

func (m *Model) helpOverlay() string {
	th := m.Theme
	title := lipgloss.NewStyle().Foreground(th.Palette.Accent).Bold(true).Render("cascade — keybindings")
	section := lipgloss.NewStyle().Foreground(th.Palette.Accent).Bold(true).Underline(true)
	keyStyle := lipgloss.NewStyle().Foreground(th.Palette.Accent).Bold(true)
	descStyle := lipgloss.NewStyle().Foreground(th.Palette.Fg)
	hintStyle := lipgloss.NewStyle().Foreground(th.Palette.Dim).Italic(true)

	row := func(k, d string) string {
		paddedKey := lipgloss.NewStyle().Width(16).Render(keyStyle.Render(k))
		return "  " + paddedKey + descStyle.Render(d)
	}

	body := strings.Join([]string{
		title,
		"",
		section.Render("NAVIGATION"),
		row("j / k / ↑ ↓", "move cursor"),
		row("l / enter", "drill into selected (or `..` to go up)"),
		row("h", "go back up"),
		row("gg / G", "top / bottom (gg jumps to `..` if shown)"),
		row("R", "refresh from disk"),
		row("ctrl+d / pgdn", "scroll details down"),
		row("ctrl+u / pgup", "scroll details up"),
		"",
		section.Render("CAPTURE & EDIT"),
		row("n", "new task at this tier"),
		row("gn", "quick-capture to inbox"),
		row("r", "rename selected"),
		row("e", "open in $EDITOR"),
		row("t", "toggle checkboxes in body ([1]…[9] overlay)"),
		"",
		section.Render("MANIPULATION"),
		row("K / J", "move up / down"),
		row("m", "move to another parent (centered fuzzy picker)"),
		row("x / space", "cycle status"),
		row("Z", "toggle hide-done (default: show, strikethrough)"),
		row("dd", "soft delete"),
		row("D", "hard delete"),
		"",
		section.Render("SEARCH & COMMANDS"),
		row("/", "filter current tier"),
		row("ctrl+f", "global fuzzy search"),
		row(":", "command palette"),
		row("?", "this help"),
		row("q / ctrl+c", "quit"),
		"",
		hintStyle.Render("tip: drill in with l, then n adds a child at that tier"),
		hintStyle.Render("types: top-level=project (■), with-children=folder (▸), leaves=task (○ ◐ ✓ ✗)"),
		hintStyle.Render("a container rolls up to ✓ when all its descendant tasks are done"),
		hintStyle.Render("override with `type: project|folder|task` in frontmatter"),
		hintStyle.Render("sort order: doing → blocked → todo → done (stable within band)"),
		hintStyle.Render("(any key to close)"),
	}, "\n")

	return lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(th.Palette.Border).
		Padding(1, 3).
		Render(body)
}
