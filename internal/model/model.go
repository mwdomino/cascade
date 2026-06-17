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

type Frontmatter struct {
	Title   string         `yaml:"title"`
	Status  Status         `yaml:"status"`
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

func (n *Node) ProgressDoneTotal() (done, total int) {
	for _, c := range n.Children {
		total++
		if c.FM.Status == StatusDone {
			done++
		}
	}
	return done, total
}
