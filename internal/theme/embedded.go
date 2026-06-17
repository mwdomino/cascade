package theme

import "embed"

//go:embed dracula.yaml gruvbox.yaml tokyonight.yaml nord.yaml
var builtinFS embed.FS

func BuiltinNames() []string {
	return []string{"dracula", "gruvbox", "tokyonight", "nord"}
}
