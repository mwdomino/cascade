package model

import "testing"

func TestStatusCycle(t *testing.T) {
	cases := []struct{ in, want Status }{
		{StatusTodo, StatusDoing},
		{StatusDoing, StatusDone},
		{StatusDone, StatusBlocked},
		{StatusBlocked, StatusTodo},
		{"", StatusTodo},
	}
	for _, c := range cases {
		if got := c.in.Cycle(); got != c.want {
			t.Errorf("%q.Cycle() = %q; want %q", c.in, got, c.want)
		}
	}
}

func TestStatusValid(t *testing.T) {
	if !StatusDoing.Valid() {
		t.Error("doing should be valid")
	}
	if Status("nope").Valid() {
		t.Error("nope should be invalid")
	}
}

func TestEffectiveTypeFallback(t *testing.T) {
	root := &Node{}
	project := &Node{Slug: "p", Parent: root}
	root.Children = []*Node{project}
	folder := &Node{Slug: "f", Parent: project}
	project.Children = []*Node{folder}
	task := &Node{Slug: "t", Parent: folder}
	folder.Children = []*Node{task}

	if got := project.EffectiveType(); got != TypeProject {
		t.Errorf("top-level: got %q, want project", got)
	}
	if got := folder.EffectiveType(); got != TypeFolder {
		t.Errorf("with children: got %q, want folder", got)
	}
	if got := task.EffectiveType(); got != TypeTask {
		t.Errorf("leaf: got %q, want task", got)
	}
}

func TestEffectivelyDoneRollup(t *testing.T) {
	root := &Node{}
	project := &Node{Slug: "p", Parent: root}
	root.Children = []*Node{project}
	folder := &Node{Slug: "f", Parent: project}
	project.Children = []*Node{folder}
	t1 := &Node{Slug: "t1", Parent: folder, FM: Frontmatter{Status: StatusDone}}
	t2 := &Node{Slug: "t2", Parent: folder, FM: Frontmatter{Status: StatusTodo}}
	folder.Children = []*Node{t1, t2}

	if project.EffectivelyDone() {
		t.Error("project should NOT be done while a leaf task is todo")
	}
	t2.FM.Status = StatusDone
	if !project.EffectivelyDone() {
		t.Error("project SHOULD be done once all descendant tasks are done")
	}

	// Empty containers are not done.
	empty := &Node{Slug: "e", Parent: root}
	if empty.EffectivelyDone() {
		t.Error("empty container should not be done")
	}
}

func TestEffectiveTypeExplicit(t *testing.T) {
	root := &Node{}
	leaf := &Node{Slug: "x", Parent: root, FM: Frontmatter{Type: TypeTask}}
	root.Children = []*Node{leaf}
	if got := leaf.EffectiveType(); got != TypeTask {
		t.Errorf("explicit type ignored: got %q", got)
	}
}

func TestNodeProgress(t *testing.T) {
	parent := &Node{Slug: "p"}
	parent.Children = []*Node{
		{Slug: "a", FM: Frontmatter{Status: StatusDone}, Parent: parent},
		{Slug: "b", FM: Frontmatter{Status: StatusTodo}, Parent: parent},
		{Slug: "c", FM: Frontmatter{Status: StatusDone}, Parent: parent},
	}
	d, total := parent.ProgressDoneTotal()
	if d != 2 || total != 3 {
		t.Errorf("got %d/%d, want 2/3", d, total)
	}
}
