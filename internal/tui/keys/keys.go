package keys

import "github.com/charmbracelet/bubbles/key"

type Map struct {
	Up, Down, In, Out, Top, Bottom    key.Binding
	Refresh, Quit                      key.Binding
	New, QuickNew, Rename, Edit        key.Binding
	StatusCycle, ToggleDone            key.Binding
	MoveUp, MoveDown, MoveTo           key.Binding
	SoftDelete, HardDelete             key.Binding
	SearchLocal, SearchGlobal, Palette key.Binding
	Help                               key.Binding
	ScrollDown, ScrollUp               key.Binding
	ToggleCheckbox                     key.Binding
}

func Default() Map {
	return Map{
		Up:           key.NewBinding(key.WithKeys("k", "up")),
		Down:         key.NewBinding(key.WithKeys("j", "down")),
		In:           key.NewBinding(key.WithKeys("l", "right", "enter")),
		Out:          key.NewBinding(key.WithKeys("h", "left", "backspace")),
		Top:          key.NewBinding(), // dispatched by gg chord handler
		Bottom:       key.NewBinding(key.WithKeys("G")),
		Refresh:      key.NewBinding(key.WithKeys("R")),
		Quit:         key.NewBinding(key.WithKeys("q", "ctrl+c")),
		New:          key.NewBinding(key.WithKeys("n")),
		QuickNew:     key.NewBinding(), // dispatched by gn chord handler
		Rename:       key.NewBinding(key.WithKeys("r")),
		Edit:         key.NewBinding(key.WithKeys("e")),
		StatusCycle:  key.NewBinding(key.WithKeys("x", " ")),
		ToggleDone:   key.NewBinding(key.WithKeys("Z")),
		MoveUp:       key.NewBinding(key.WithKeys("K")),
		MoveDown:     key.NewBinding(key.WithKeys("J")),
		MoveTo:       key.NewBinding(key.WithKeys("m")),
		SoftDelete:   key.NewBinding(key.WithKeys("d", "d")),
		HardDelete:   key.NewBinding(key.WithKeys("D")),
		SearchLocal:  key.NewBinding(key.WithKeys("/")),
		SearchGlobal: key.NewBinding(key.WithKeys("ctrl+f")),
		Palette:      key.NewBinding(key.WithKeys(":")),
		Help:         key.NewBinding(key.WithKeys("?")),
		ScrollDown:   key.NewBinding(key.WithKeys("ctrl+d", "pgdown")),
		ScrollUp:     key.NewBinding(key.WithKeys("ctrl+u", "pgup")),
		ToggleCheckbox: key.NewBinding(key.WithKeys("t")),
	}
}
