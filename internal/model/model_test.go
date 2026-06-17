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
