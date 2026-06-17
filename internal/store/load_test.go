package store

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/mwdomino/cascade/internal/model"
)

func writeNode(t *testing.T, dir string, fm model.Frontmatter, body string) {
	t.Helper()
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := WriteIndex(filepath.Join(dir, "index.md"), fm, body); err != nil {
		t.Fatal(err)
	}
}

func TestLoadTree(t *testing.T) {
	tasks := t.TempDir()
	writeNode(t, filepath.Join(tasks, "010-work"),
		model.Frontmatter{Title: "Work"}, "")
	writeNode(t, filepath.Join(tasks, "010-work", "010-ship"),
		model.Frontmatter{Title: "Ship v1", Status: model.StatusDoing}, "body")
	writeNode(t, filepath.Join(tasks, "010-work", "020-fix"),
		model.Frontmatter{Title: "Fix bug", Status: model.StatusTodo}, "")
	writeNode(t, filepath.Join(tasks, "020-personal"),
		model.Frontmatter{Title: "Personal"}, "")
	// .trash and dotfiles must be ignored
	if err := os.MkdirAll(filepath.Join(tasks, ".trash", "junk"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(tasks, "no-prefix-dir"), 0o755); err != nil {
		t.Fatal(err)
	}

	tree, err := Load(tasks)
	if err != nil {
		t.Fatal(err)
	}
	if len(tree.Root.Children) != 2 {
		t.Fatalf("want 2 top-level, got %d", len(tree.Root.Children))
	}
	work := tree.Root.Children[0]
	if work.Slug != "work" || work.Prefix != 10 {
		t.Errorf("work parsed wrong: %+v", work)
	}
	if len(work.Children) != 2 {
		t.Fatalf("want 2 children of work, got %d", len(work.Children))
	}
	if work.Children[0].Slug != "ship" || work.Children[1].Slug != "fix" {
		t.Errorf("siblings out of order: %v",
			[]string{work.Children[0].Slug, work.Children[1].Slug})
	}
}
