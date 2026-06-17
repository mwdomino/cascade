package store

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/mwdomino/cascade/internal/model"
)

func TestRoundTripFrontmatter(t *testing.T) {
	dir := t.TempDir()
	p := filepath.Join(dir, "index.md")

	in := model.Frontmatter{
		Title:   "Hello",
		Status:  model.StatusDoing,
		Created: time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC),
		Tags:    []string{"a", "b"},
		Extra:   map[string]any{"github_repo": "foo/bar"},
	}
	if err := WriteIndex(p, in, "body line 1\nbody line 2\n"); err != nil {
		t.Fatal(err)
	}
	fm, body, err := ReadIndex(p)
	if err != nil {
		t.Fatal(err)
	}
	if fm.Title != "Hello" || fm.Status != model.StatusDoing {
		t.Errorf("metadata mismatch: %+v", fm)
	}
	if fm.Extra["github_repo"] != "foo/bar" {
		t.Errorf("extra not preserved: %+v", fm.Extra)
	}
	if body != "body line 1\nbody line 2\n" {
		t.Errorf("body mismatch: %q", body)
	}
	if fm.Updated.IsZero() {
		t.Error("Updated should be set on write")
	}
}

func TestReadIndexMissingFrontmatter(t *testing.T) {
	dir := t.TempDir()
	p := filepath.Join(dir, "index.md")
	if err := os.WriteFile(p, []byte("just a body\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	fm, body, err := ReadIndex(p)
	if err != nil {
		t.Fatal(err)
	}
	if body != "just a body\n" {
		t.Errorf("body mismatch: %q", body)
	}
	if fm.Status != "" {
		t.Errorf("expected empty status, got %q", fm.Status)
	}
}
