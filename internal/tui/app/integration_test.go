package app

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/x/exp/teatest"
	"github.com/mwdomino/cascade/internal/action"
	"github.com/mwdomino/cascade/internal/config"
	"github.com/mwdomino/cascade/internal/store"
	"github.com/mwdomino/cascade/internal/theme"
)

func TestIntegrationGoldenPath(t *testing.T) {
	// Copy fixture into a tmpdir so the test never mutates the repo.
	src := filepath.Join("..", "..", "..", "testdata", "fixtures", "basic")
	dst := t.TempDir()
	if err := copyDir(src, dst); err != nil {
		t.Fatal(err)
	}
	tree, err := store.Load(dst)
	if err != nil {
		t.Fatal(err)
	}
	th, _ := theme.Resolve(&config.Config{ThemeName: "dracula"})
	cfg := &config.Config{TasksDir: dst, Inbox: "999-inbox"}
	reg := action.NewRegistry(map[string]config.ActionDef{
		"echo-title": {Cmd: `echo "$CASCADE_TITLE"`},
	})

	tm := teatest.NewTestModel(t, New(tree, th, cfg, reg),
		teatest.WithInitialTermSize(120, 40))

	// Drill into Work, drill into Ship, back, back, create a new sibling, quit.
	send := func(r rune) { tm.Send(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{r}}) }
	send('l') // → work
	send('l') // → ship v1
	send('h') // ← work
	send('h') // ← root
	send('n')
	tm.Type("Smoke Test Task")
	tm.Send(tea.KeyMsg{Type: tea.KeyEnter})
	send('q')
	tm.WaitFinished(t, teatest.WithFinalTimeout(3*time.Second))

	// Verify on disk
	entries, _ := os.ReadDir(dst)
	found := false
	for _, e := range entries {
		if e.IsDir() && filepath.Base(e.Name()) != ".trash" {
			t.Logf("found entry: %s", e.Name())
			if strings.Contains(e.Name(), "smoke-test-task") {
				found = true
			}
		}
	}
	if !found {
		t.Error("smoke-test-task not present on disk")
	}
}

func copyDir(src, dst string) error {
	return filepath.Walk(src, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		rel, _ := filepath.Rel(src, path)
		target := filepath.Join(dst, rel)
		if info.IsDir() {
			return os.MkdirAll(target, 0o755)
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		return os.WriteFile(target, data, 0o644)
	})
}
