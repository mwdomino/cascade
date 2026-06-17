package editor

import (
	"os"
	"path/filepath"
	"testing"
)

func TestOpenInvokesEditor(t *testing.T) {
	dir := t.TempDir()
	target := filepath.Join(dir, "file.md")
	if err := os.WriteFile(target, []byte("hello"), 0o644); err != nil {
		t.Fatal(err)
	}
	// Use `true` as a no-op "editor" so the test runs in CI without a TTY.
	t.Setenv("EDITOR", "true")
	if err := Open(target); err != nil {
		t.Errorf("Open returned error: %v", err)
	}
}

func TestOpenEditorFailure(t *testing.T) {
	t.Setenv("EDITOR", "false")
	if err := Open("/tmp/does-not-matter"); err == nil {
		t.Error("expected error when editor exits non-zero")
	}
}
