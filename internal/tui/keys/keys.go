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
}

func Default() Map {
	return Map{
		Up:           key.NewBinding(key.WithKeys("k", "up")),
		Down:         key.NewBinding(key.WithKeys("j", "down")),
		In:           key.NewBinding(key.WithKeys("l", "right", "enter")),
		Out:          key.NewBinding(key.WithKeys("h", "left")),
		Top:          key.NewBinding(key.WithKeys("g", "g")),
		Bottom:       key.NewBinding(key.WithKeys("G")),
		Refresh:      key.NewBinding(key.WithKeys("R")),
		Quit:         key.NewBinding(key.WithKeys("q", "ctrl+c")),
		New:          key.NewBinding(key.WithKeys("n")),
		QuickNew:     key.NewBinding(key.WithKeys("g", "n")),
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
	}
}
