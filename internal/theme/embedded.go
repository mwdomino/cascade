package theme

import "embed"

//go:embed dracula.yaml
var builtinFS embed.FS

func builtinNames() []string { return []string{"dracula"} }
