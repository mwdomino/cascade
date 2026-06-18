package store

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/mwdomino/cascade/internal/model"
)

func newTree(t *testing.T) *Tree {
	t.Helper()
	dir := t.TempDir()
	tree, err := Load(dir)
	if err != nil {
		t.Fatal(err)
	}
	return tree
}

func TestCreate(t *testing.T) {
	tree := newTree(t)
	n, err := tree.Create(tree.Root, "First Task")
	if err != nil {
		t.Fatal(err)
	}
	if n.Prefix != 10 || n.Slug != "first-task" {
		t.Errorf("got prefix=%d slug=%q", n.Prefix, n.Slug)
	}
	if n.FM.Status != model.StatusTodo {
		t.Errorf("default status %q want todo", n.FM.Status)
	}
	// Second sibling = 020-
	n2, _ := tree.Create(tree.Root, "Second")
	if n2.Prefix != 20 {
		t.Errorf("got prefix=%d want 20", n2.Prefix)
	}
}

func TestRename(t *testing.T) {
	tree := newTree(t)
	n, _ := tree.Create(tree.Root, "old name")
	oldPath := n.Path
	if err := tree.Rename(n, "Brand New"); err != nil {
		t.Fatal(err)
	}
	if n.Slug != "brand-new" {
		t.Errorf("slug %q want brand-new", n.Slug)
	}
	if _, err := os.Stat(oldPath); !os.IsNotExist(err) {
		t.Errorf("old path still exists: %v", err)
	}
	if !strings.HasSuffix(n.Path, "010-brand-new") {
		t.Errorf("new path: %s", n.Path)
	}
}

func TestMoveUpDown(t *testing.T) {
	tree := newTree(t)
	a, _ := tree.Create(tree.Root, "A")
	b, _ := tree.Create(tree.Root, "B")
	if err := tree.MoveUp(b); err != nil {
		t.Fatal(err)
	}
	if a.Prefix != 20 || b.Prefix != 10 {
		t.Errorf("after MoveUp(b): a=%d b=%d", a.Prefix, b.Prefix)
	}
}

func TestSoftDelete(t *testing.T) {
	tree := newTree(t)
	n, _ := tree.Create(tree.Root, "doomed")
	if err := tree.SoftDelete(n); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(n.Path); !os.IsNotExist(err) {
		t.Errorf("source still exists")
	}
	trash := filepath.Join(tree.TasksDir, ".trash")
	entries, _ := os.ReadDir(trash)
	if len(entries) != 1 {
		t.Fatalf("trash entries: %d", len(entries))
	}
	if !strings.Contains(entries[0].Name(), "doomed") {
		t.Errorf("trash name: %s", entries[0].Name())
	}
}

func TestCreateAtRespectsPrefix(t *testing.T) {
	tree := newTree(t)
	tree.Create(tree.Root, "first")
	tree.Create(tree.Root, "second")
	n, err := tree.CreateAt(tree.Root, "inbox", 999)
	if err != nil {
		t.Fatal(err)
	}
	if n.Prefix != 999 {
		t.Errorf("expected prefix 999, got %d", n.Prefix)
	}
	if !strings.HasSuffix(n.Path, "999-inbox") {
		t.Errorf("path should end with 999-inbox, got %s", n.Path)
	}
}

func TestCreateAtCollisionFallsBack(t *testing.T) {
	tree := newTree(t)
	tree.CreateAt(tree.Root, "a", 100)
	n, err := tree.CreateAt(tree.Root, "b", 100)
	if err != nil {
		t.Fatal(err)
	}
	if n.Prefix == 100 {
		t.Errorf("expected fallback prefix, got the colliding one (100)")
	}
}

func TestSetStatus(t *testing.T) {
	tree := newTree(t)
	n, _ := tree.Create(tree.Root, "task")
	if err := tree.SetStatus(n, model.StatusDoing); err != nil {
		t.Fatal(err)
	}
	if n.FM.Status != model.StatusDoing {
		t.Errorf("status %q", n.FM.Status)
	}
	fm, _, _ := ReadIndex(filepath.Join(n.Path, "index.md"))
	if fm.Status != model.StatusDoing {
		t.Errorf("disk status %q", fm.Status)
	}
}

func TestAllNodes(t *testing.T) {
	tree := newTree(t)
	a, _ := tree.Create(tree.Root, "A")
	tree.Create(a, "A1")
	tree.Create(tree.Root, "B")
	if len(tree.AllNodes()) != 3 {
		t.Errorf("got %d nodes", len(tree.AllNodes()))
	}
}

func TestSoftDeletePurgesDescendants(t *testing.T) {
	tree := newTree(t)
	parent, _ := tree.Create(tree.Root, "parent")
	child, _ := tree.Create(parent, "child")
	childPath := child.Path
	if err := tree.SoftDelete(parent); err != nil {
		t.Fatal(err)
	}
	if n := tree.NodeAt(childPath); n != nil {
		t.Errorf("child path still in byPath: %s", childPath)
	}
}
