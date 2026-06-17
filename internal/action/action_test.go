package action

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/mwdomino/cascade/internal/config"
	"github.com/mwdomino/cascade/internal/model"
)

func node(t *testing.T) *model.Node {
	t.Helper()
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "index.md"), []byte("body"), 0o644); err != nil {
		t.Fatal(err)
	}
	return &model.Node{
		Path: dir,
		Slug: "task",
		FM: model.Frontmatter{
			Title:  "My Task",
			Status: model.StatusDoing,
			Tags:   []string{"a", "b"},
			Extra:  map[string]any{"github_repo": "foo/bar"},
		},
	}
}

func TestActionEnvInjection(t *testing.T) {
	defs := map[string]config.ActionDef{
		"echo-env": {
			Cmd: `echo "title=$CASCADE_TITLE repo=$CASCADE_FM_GITHUB_REPO tags=$CASCADE_TAGS"`,
		},
	}
	reg := NewRegistry(defs)
	acts := reg.Applicable(node(t))
	if len(acts) != 1 {
		t.Fatalf("got %d actions", len(acts))
	}
	res, err := acts[0].Run(node(t))
	if err != nil {
		t.Fatalf("run: %v out=%q err=%q", err, res.Stdout, res.Stderr)
	}
	if !strings.Contains(res.Stdout, "title=My Task") ||
		!strings.Contains(res.Stdout, "repo=foo/bar") ||
		!strings.Contains(res.Stdout, "tags=a b") {
		t.Errorf("env not injected: %q", res.Stdout)
	}
}

func TestActionWhenGating(t *testing.T) {
	defs := map[string]config.ActionDef{
		"needs-repo": {
			Cmd:  "true",
			When: config.ActionWhen{HasFrontmatter: []string{"github_repo"}},
		},
		"needs-jira": {
			Cmd:  "true",
			When: config.ActionWhen{HasFrontmatter: []string{"jira_ticket"}},
		},
	}
	reg := NewRegistry(defs)
	names := map[string]bool{}
	for _, a := range reg.Applicable(node(t)) {
		names[a.Name] = true
	}
	if !names["needs-repo"] {
		t.Error("needs-repo should be applicable")
	}
	if names["needs-jira"] {
		t.Error("needs-jira should not be applicable")
	}
}
