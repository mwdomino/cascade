package main

import (
	"fmt"
	"os"
	"runtime/debug"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/mwdomino/cascade/internal/action"
	"github.com/mwdomino/cascade/internal/config"
	"github.com/mwdomino/cascade/internal/store"
	"github.com/mwdomino/cascade/internal/theme"
	"github.com/mwdomino/cascade/internal/tui/app"
)

// version is overridden at release time via -ldflags "-X main.version=…"
// (see .goreleaser.yaml). For dev / `go install` builds it falls through
// to resolveVersion() which consults runtime/debug.ReadBuildInfo.
var version = "dev"

func main() {
	if len(os.Args) > 1 {
		switch os.Args[1] {
		case "version", "-v", "-version", "--version":
			fmt.Printf("cascade %s\n", resolveVersion())
			return
		case "-h", "-help", "--help", "help":
			printUsage()
			return
		}
	}

	cfg, err := config.Load()
	if err != nil {
		fmt.Fprintln(os.Stderr, "config:", err)
		os.Exit(1)
	}
	if err := os.MkdirAll(cfg.TasksDir, 0o755); err != nil {
		fmt.Fprintln(os.Stderr, "tasks_dir:", err)
		os.Exit(1)
	}
	tree, err := store.Load(cfg.TasksDir)
	if err != nil {
		fmt.Fprintln(os.Stderr, "load:", err)
		os.Exit(1)
	}
	th, err := theme.Resolve(cfg)
	if err != nil {
		fmt.Fprintln(os.Stderr, "theme:", err)
		os.Exit(1)
	}
	app.Version = resolveVersion()
	reg := action.NewRegistry(cfg.Actions)
	p := tea.NewProgram(app.New(tree, th, cfg, reg), tea.WithAltScreen())
	if _, err := p.Run(); err != nil {
		fmt.Fprintln(os.Stderr, "tui:", err)
		os.Exit(1)
	}
}

// resolveVersion reports the build identifier in priority order:
//  1. The goreleaser-injected `version` ldflag.
//  2. The module version embedded by `go install path@vX.Y.Z`.
//  3. `dev-<short-revision>[-dirty]` from runtime/debug.ReadBuildInfo,
//     so `just build` and `go build` from source still show something
//     traceable instead of a hard-coded placeholder.
func resolveVersion() string {
	if version != "" && version != "dev" {
		return version
	}
	bi, ok := debug.ReadBuildInfo()
	if !ok {
		return "dev"
	}
	if bi.Main.Version != "" && bi.Main.Version != "(devel)" {
		return bi.Main.Version
	}
	var rev string
	var dirty bool
	for _, s := range bi.Settings {
		switch s.Key {
		case "vcs.revision":
			rev = s.Value
		case "vcs.modified":
			if s.Value == "true" {
				dirty = true
			}
		}
	}
	if rev == "" {
		return "dev"
	}
	if len(rev) > 7 {
		rev = rev[:7]
	}
	out := "dev-" + rev
	if dirty {
		out += "-dirty"
	}
	return out
}

func printUsage() {
	fmt.Fprintln(os.Stderr, "usage: cascade [command]")
	fmt.Fprintln(os.Stderr, "")
	fmt.Fprintln(os.Stderr, "  (no args)   launch the TUI")
	fmt.Fprintln(os.Stderr, "  version     print the build version and exit")
	fmt.Fprintln(os.Stderr, "  help        show this message")
	fmt.Fprintln(os.Stderr, "")
	fmt.Fprintln(os.Stderr, "config:")
	fmt.Fprintln(os.Stderr, "  $XDG_CONFIG_HOME/cascade/config.yaml (global)")
	fmt.Fprintln(os.Stderr, "  $PWD/.cascade.yaml                   (project override)")
}
