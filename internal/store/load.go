package store

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/mwdomino/cascade/internal/model"
)

type Tree struct {
	Root     *model.Node
	TasksDir string
	byPath   map[string]*model.Node
}

func Load(tasksDir string) (*Tree, error) {
	abs, err := filepath.Abs(tasksDir)
	if err != nil {
		return nil, err
	}
	root := &model.Node{Path: abs, Slug: ""}
	tree := &Tree{Root: root, TasksDir: abs, byPath: map[string]*model.Node{abs: root}}

	if _, err := os.Stat(abs); os.IsNotExist(err) {
		return tree, nil // empty vault is fine
	}
	if err := loadChildren(root, tree.byPath); err != nil {
		return nil, err
	}
	return tree, nil
}

func loadChildren(parent *model.Node, byPath map[string]*model.Node) error {
	entries, err := os.ReadDir(parent.Path)
	if err != nil {
		return fmt.Errorf("read dir %s: %w", parent.Path, err)
	}
	var nodes []*model.Node
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		name := e.Name()
		if strings.HasPrefix(name, ".") {
			continue
		}
		prefix, slug, ok := ParsePrefix(name)
		if !ok {
			continue
		}
		child := &model.Node{
			Path:   filepath.Join(parent.Path, name),
			Prefix: prefix,
			Slug:   slug,
			Parent: parent,
		}
		indexPath := filepath.Join(child.Path, "index.md")
		if _, err := os.Stat(indexPath); err == nil {
			fm, body, err := ReadIndex(indexPath)
			if err != nil {
				return err
			}
			child.FM = fm
			child.Body = body
		}
		nodes = append(nodes, child)
		byPath[child.Path] = child
	}
	sort.SliceStable(nodes, func(i, j int) bool {
		return nodes[i].Prefix < nodes[j].Prefix
	})
	parent.Children = nodes
	for _, n := range nodes {
		if err := loadChildren(n, byPath); err != nil {
			return err
		}
	}
	return nil
}

func (t *Tree) NodeAt(path string) *model.Node {
	abs, err := filepath.Abs(path)
	if err != nil {
		return nil
	}
	return t.byPath[abs]
}
