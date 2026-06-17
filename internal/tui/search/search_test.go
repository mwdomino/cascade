package search

import (
	"testing"

	"github.com/mwdomino/cascade/internal/model"
)

func n(title string, tags []string, body string) *model.Node {
	return &model.Node{FM: model.Frontmatter{Title: title, Tags: tags}, Body: body}
}

func TestLocalFilter(t *testing.T) {
	items := []*model.Node{
		n("write readme", nil, ""),
		n("fix bug", []string{"backend"}, ""),
		n("WRITE tests", nil, ""),
	}
	got := LocalFilter(items, "write")
	if len(got) != 2 {
		t.Errorf("got %d, want 2", len(got))
	}
}

func TestGlobalFuzzy(t *testing.T) {
	items := []*model.Node{
		n("Ship cascade v1", nil, "release notes go here"),
		n("Auth refactor", []string{"auth"}, ""),
		n("Bug triage", nil, "authentication issue"),
	}
	got := GlobalFuzzy(items, "auth")
	if len(got) < 2 {
		t.Errorf("expected at least 2 fuzzy matches, got %d", len(got))
	}
}
