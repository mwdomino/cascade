package store

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/mwdomino/cascade/internal/model"
)

var slugRe = regexp.MustCompile(`[^a-z0-9]+`)

func slugify(s string) string {
	s = strings.ToLower(strings.TrimSpace(s))
	s = slugRe.ReplaceAllString(s, "-")
	s = strings.Trim(s, "-")
	if s == "" {
		s = "untitled"
	}
	return s
}

func (t *Tree) nextPrefix(parent *model.Node) int {
	if len(parent.Children) == 0 {
		return 10
	}
	return parent.Children[len(parent.Children)-1].Prefix + 10
}

func (t *Tree) Create(parent *model.Node, title string) (*model.Node, error) {
	if parent == nil {
		parent = t.Root
	}
	prefix := t.nextPrefix(parent)
	slug := slugify(title)
	dir := filepath.Join(parent.Path, FormatPrefix(prefix, slug))
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return nil, err
	}
	fm := model.Frontmatter{Title: title, Status: model.StatusTodo}
	if err := WriteIndex(filepath.Join(dir, "index.md"), fm, ""); err != nil {
		return nil, err
	}
	child := &model.Node{
		Path:   dir,
		Prefix: prefix,
		Slug:   slug,
		FM:     fm,
		Parent: parent,
	}
	parent.Children = append(parent.Children, child)
	t.byPath[dir] = child
	// Reload to pick up any disk-side details (e.g. Created timestamp).
	if reloadFM, body, err := ReadIndex(filepath.Join(dir, "index.md")); err == nil {
		child.FM = reloadFM
		child.Body = body
	}
	return child, nil
}

func (t *Tree) Rename(n *model.Node, newTitle string) error {
	newSlug := slugify(newTitle)
	if newSlug == n.Slug {
		n.FM.Title = newTitle
		return WriteIndex(filepath.Join(n.Path, "index.md"), n.FM, n.Body)
	}
	newPath := filepath.Join(filepath.Dir(n.Path), FormatPrefix(n.Prefix, newSlug))
	if err := os.Rename(n.Path, newPath); err != nil {
		return err
	}
	delete(t.byPath, n.Path)
	n.Path = newPath
	n.Slug = newSlug
	n.FM.Title = newTitle
	t.byPath[newPath] = n
	// Update descendant byPath entries
	t.rebuildByPathSubtree(n)
	return WriteIndex(filepath.Join(n.Path, "index.md"), n.FM, n.Body)
}

func (t *Tree) rebuildByPathSubtree(root *model.Node) {
	for _, c := range root.Children {
		newChildPath := filepath.Join(root.Path, FormatPrefix(c.Prefix, c.Slug))
		delete(t.byPath, c.Path)
		c.Path = newChildPath
		t.byPath[c.Path] = c
		t.rebuildByPathSubtree(c)
	}
}

func (t *Tree) SetStatus(n *model.Node, s model.Status) error {
	n.FM.Status = s
	return WriteIndex(filepath.Join(n.Path, "index.md"), n.FM, n.Body)
}

func (t *Tree) swapWithNeighbor(n *model.Node, dir int) error {
	if n.Parent == nil {
		return fmt.Errorf("root has no siblings")
	}
	siblings := n.Parent.Children
	idx := -1
	for i, s := range siblings {
		if s == n {
			idx = i
			break
		}
	}
	if idx < 0 {
		return fmt.Errorf("node not found in parent")
	}
	target := idx + dir
	if target < 0 || target >= len(siblings) {
		return nil // at edge, no-op
	}
	other := siblings[target]
	n.Prefix, other.Prefix = other.Prefix, n.Prefix
	if err := t.renameWithCurrentPrefix(n); err != nil {
		return err
	}
	if err := t.renameWithCurrentPrefix(other); err != nil {
		return err
	}
	siblings[idx], siblings[target] = siblings[target], siblings[idx]
	return nil
}

func (t *Tree) renameWithCurrentPrefix(n *model.Node) error {
	newPath := filepath.Join(filepath.Dir(n.Path), FormatPrefix(n.Prefix, n.Slug))
	if newPath == n.Path {
		return nil
	}
	if err := os.Rename(n.Path, newPath); err != nil {
		return err
	}
	delete(t.byPath, n.Path)
	n.Path = newPath
	t.byPath[newPath] = n
	t.rebuildByPathSubtree(n)
	return nil
}

func (t *Tree) MoveUp(n *model.Node) error   { return t.swapWithNeighbor(n, -1) }
func (t *Tree) MoveDown(n *model.Node) error { return t.swapWithNeighbor(n, +1) }

func (t *Tree) MoveTo(n *model.Node, newParent *model.Node) error {
	if newParent == nil {
		newParent = t.Root
	}
	if newParent == n {
		return fmt.Errorf("cannot move into self")
	}
	for p := newParent; p != nil; p = p.Parent {
		if p == n {
			return fmt.Errorf("cannot move into own descendant")
		}
	}
	newPrefix := t.nextPrefix(newParent)
	newPath := filepath.Join(newParent.Path, FormatPrefix(newPrefix, n.Slug))
	if err := os.Rename(n.Path, newPath); err != nil {
		return err
	}
	// Detach from old parent
	if n.Parent != nil {
		for i, s := range n.Parent.Children {
			if s == n {
				n.Parent.Children = append(n.Parent.Children[:i], n.Parent.Children[i+1:]...)
				break
			}
		}
	}
	delete(t.byPath, n.Path)
	n.Path = newPath
	n.Prefix = newPrefix
	n.Parent = newParent
	newParent.Children = append(newParent.Children, n)
	t.byPath[newPath] = n
	t.rebuildByPathSubtree(n)
	return nil
}

func (t *Tree) SoftDelete(n *model.Node) error {
	if n.Parent == nil {
		return fmt.Errorf("cannot delete root")
	}
	trashDir := filepath.Join(t.TasksDir, ".trash")
	if err := os.MkdirAll(trashDir, 0o755); err != nil {
		return err
	}
	ts := time.Now().UTC().Format("20060102T150405")
	dest := filepath.Join(trashDir, fmt.Sprintf("%s-%s", ts, n.Slug))
	if err := os.Rename(n.Path, dest); err != nil {
		return err
	}
	t.detach(n)
	return nil
}

func (t *Tree) HardDelete(n *model.Node) error {
	if n.Parent == nil {
		return fmt.Errorf("cannot delete root")
	}
	if err := os.RemoveAll(n.Path); err != nil {
		return err
	}
	t.detach(n)
	return nil
}

func (t *Tree) detach(n *model.Node) {
	if n.Parent != nil {
		for i, s := range n.Parent.Children {
			if s == n {
				n.Parent.Children = append(n.Parent.Children[:i], n.Parent.Children[i+1:]...)
				break
			}
		}
	}
	t.purgeByPath(n)
}

func (t *Tree) purgeByPath(n *model.Node) {
	delete(t.byPath, n.Path)
	for _, c := range n.Children {
		t.purgeByPath(c)
	}
}

func (t *Tree) Reload() error {
	newTree, err := Load(t.TasksDir)
	if err != nil {
		return err
	}
	t.Root = newTree.Root
	t.byPath = newTree.byPath
	return nil
}
