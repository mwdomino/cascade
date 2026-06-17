package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestMergeProjectOverridesGlobal(t *testing.T) {
	global := t.TempDir()
	project := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", global)
	t.Setenv("HOME", t.TempDir()) // isolate
	wd, _ := os.Getwd()
	defer os.Chdir(wd)
	if err := os.Chdir(project); err != nil {
		t.Fatal(err)
	}

	globalDir := filepath.Join(global, "cascade")
	os.MkdirAll(globalDir, 0o755)
	os.WriteFile(filepath.Join(globalDir, "config.yaml"), []byte(`
tasks_dir: /global/path
theme: dracula
actions:
  global-only:
    cmd: 'echo global'
`), 0o644)
	os.WriteFile(filepath.Join(project, ".cascade.yaml"), []byte(`
tasks_dir: /project/path
actions:
  project-only:
    cmd: 'echo project'
`), 0o644)

	cfg, err := Load()
	if err != nil {
		t.Fatal(err)
	}
	if cfg.TasksDir != "/project/path" {
		t.Errorf("tasks_dir = %q", cfg.TasksDir)
	}
	if cfg.ThemeName != "dracula" {
		t.Errorf("theme = %q", cfg.ThemeName)
	}
	if _, ok := cfg.Actions["global-only"]; !ok {
		t.Error("global-only action missing")
	}
	if _, ok := cfg.Actions["project-only"]; !ok {
		t.Error("project-only action missing")
	}
}

func TestDefaults(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("XDG_CONFIG_HOME", filepath.Join(home, ".config"))
	t.Chdir(t.TempDir())

	cfg, err := Load()
	if err != nil {
		t.Fatal(err)
	}
	want := filepath.Join(home, ".cascade")
	if cfg.TasksDir != want {
		t.Errorf("tasks_dir = %q, want %q", cfg.TasksDir, want)
	}
	if cfg.Inbox != "999-inbox" {
		t.Errorf("inbox = %q", cfg.Inbox)
	}
}
