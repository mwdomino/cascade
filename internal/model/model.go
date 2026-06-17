package model

import "time"

type Status string

const (
	StatusTodo    Status = "todo"
	StatusDoing   Status = "doing"
	StatusDone    Status = "done"
	StatusBlocked Status = "blocked"
)

func (s Status) Valid() bool {
	switch s {
	case StatusTodo, StatusDoing, StatusDone, StatusBlocked:
		return true
	}
	return false
}

func (s Status) Cycle() Status {
	switch s {
	case StatusTodo:
		return StatusDoing
	case StatusDoing:
		return StatusDone
	case StatusDone:
		return StatusBlocked
	case StatusBlocked:
		return StatusTodo
	default:
		return StatusTodo
	}
}

type NodeType string

const (
	TypeProject NodeType = "project"
	TypeFolder  NodeType = "folder"
	TypeTask    NodeType = "task"
)

func (t NodeType) Valid() bool {
	switch t {
	case TypeProject, TypeFolder, TypeTask:
		return true
	}
	return false
}

type Frontmatter struct {
	Title   string         `yaml:"title"`
	Status  Status         `yaml:"status"`
	Type    NodeType       `yaml:"type,omitempty"`
	Created time.Time      `yaml:"created"`
	Updated time.Time      `yaml:"updated"`
	Tags    []string       `yaml:"tags,omitempty"`
	Extra   map[string]any `yaml:",inline"`
}

type Node struct {
	Path     string      // absolute path to the folder
	Prefix   int         // numeric prefix (010, 020, ...)
	Slug     string      // folder name without prefix
	FM       Frontmatter
	Body     string
	Parent   *Node
	Children []*Node
}

func (n *Node) Title() string {
	if n.FM.Title != "" {
		return n.FM.Title
	}
	return n.Slug
}

func (n *Node) IsRoot() bool { return n.Parent == nil }

func (n *Node) Depth() int {
	d := 0
	for cur := n.Parent; cur != nil; cur = cur.Parent {
		d++
	}
	return d
}

// EffectiveType returns the explicit Frontmatter.Type if set, otherwise
// derives a sensible default from position and shape:
//   - top-level with children → project
//   - non-top-level with children → folder
//   - leaf (any depth) → task
//
// A top-level leaf is a task, not a project, so the user can mark it done
// with x. Once it sprouts children it becomes a project automatically.
// The synthetic root itself returns TypeFolder.
func (n *Node) EffectiveType() NodeType {
	if n.FM.Type.Valid() {
		return n.FM.Type
	}
	if n.IsRoot() {
		return TypeFolder
	}
	if len(n.Children) == 0 {
		return TypeTask
	}
	if n.Parent != nil && n.Parent.IsRoot() {
		return TypeProject
	}
	return TypeFolder
}

func (n *Node) IsContainer() bool {
	t := n.EffectiveType()
	return t == TypeProject || t == TypeFolder
}

// EffectivelyDone reports whether this node is "done" from the user's POV.
// A task is done when its status is StatusDone. A container is done when it
// has at least one child and every direct child is EffectivelyDone — which
// recursively requires every descendant task to be done.
func (n *Node) EffectivelyDone() bool {
	if n.EffectiveType() == TypeTask {
		return n.FM.Status == StatusDone
	}
	if len(n.Children) == 0 {
		return false
	}
	for _, c := range n.Children {
		if !c.EffectivelyDone() {
			return false
		}
	}
	return true
}

func (n *Node) ProgressDoneTotal() (done, total int) {
	for _, c := range n.Children {
		total++
		if c.EffectivelyDone() {
			done++
		}
	}
	return done, total
}
