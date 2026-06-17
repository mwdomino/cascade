package main

import (
	"flag"
	"fmt"
	"os"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/mwdomino/cascade/internal/config"
	"github.com/mwdomino/cascade/internal/store"
	"github.com/mwdomino/cascade/internal/theme"
	"github.com/mwdomino/cascade/internal/tui/app"
)

var version = "0.0.1-dev"

func main() {
	showVersion := flag.Bool("version", false, "print version and exit")
	flag.Parse()
	if *showVersion {
		fmt.Printf("cascade %s\n", version)
		return
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
	p := tea.NewProgram(app.New(tree, th, cfg), tea.WithAltScreen())
	if _, err := p.Run(); err != nil {
		fmt.Fprintln(os.Stderr, "tui:", err)
		os.Exit(1)
	}
}
