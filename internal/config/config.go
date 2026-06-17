package config

import (
	"errors"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
)

type Config struct {
	TasksDir    string               `yaml:"tasks_dir"`
	Inbox       string               `yaml:"inbox"`
	ThemeName   string               `yaml:"-"`
	ThemeInline map[string]any       `yaml:"-"`
	Actions     map[string]ActionDef `yaml:"actions"`

	Raw struct {
		Theme any `yaml:"theme"`
	} `yaml:",inline"`
}

type ActionDef struct {
	Cmd     string     `yaml:"cmd"`
	Stdin   string     `yaml:"stdin"`
	Keybind string     `yaml:"keybind"`
	When    ActionWhen `yaml:"when"`
}

type ActionWhen struct {
	HasFrontmatter []string `yaml:"has_frontmatter"`
}

func Load() (*Config, error) {
	cfg := &Config{
		TasksDir: defaultTasksDir(),
		Inbox:    "999-inbox",
		Actions:  map[string]ActionDef{},
	}
	if globalPath, ok := globalConfigPath(); ok {
		if err := mergeFile(cfg, globalPath); err != nil {
			return nil, err
		}
	}
	if _, err := os.Stat(".cascade.yaml"); err == nil {
		if err := mergeFile(cfg, ".cascade.yaml"); err != nil {
			return nil, err
		}
	}
	// expand ~
	cfg.TasksDir = expandHome(cfg.TasksDir)
	return cfg, nil
}

func defaultTasksDir() string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".cascade")
}

func globalConfigPath() (string, bool) {
	xdg := os.Getenv("XDG_CONFIG_HOME")
	if xdg == "" {
		home, _ := os.UserHomeDir()
		xdg = filepath.Join(home, ".config")
	}
	p := filepath.Join(xdg, "cascade", "config.yaml")
	if _, err := os.Stat(p); err != nil {
		return "", false
	}
	return p, true
}

func expandHome(p string) string {
	if strings.HasPrefix(p, "~/") {
		home, _ := os.UserHomeDir()
		return filepath.Join(home, p[2:])
	}
	return p
}

func mergeFile(cfg *Config, path string) error {
	raw, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	tmp := &Config{Actions: map[string]ActionDef{}}
	if err := yaml.Unmarshal(raw, tmp); err != nil {
		return err
	}
	if tmp.TasksDir != "" {
		cfg.TasksDir = tmp.TasksDir
	}
	if tmp.Inbox != "" {
		cfg.Inbox = tmp.Inbox
	}
	switch v := tmp.Raw.Theme.(type) {
	case string:
		cfg.ThemeName = v
		cfg.ThemeInline = nil
	case map[string]any:
		cfg.ThemeName = ""
		cfg.ThemeInline = v
	case nil:
		// unchanged
	default:
		return errors.New("invalid theme: must be a name (string) or an inline map")
	}
	for k, v := range tmp.Actions {
		cfg.Actions[k] = v
	}
	return nil
}
