package action

import (
	"bytes"
	"fmt"
	"os/exec"
	"path/filepath"
	"regexp"
	"sort"
	"strings"

	"github.com/mwdomino/cascade/internal/config"
	"github.com/mwdomino/cascade/internal/model"
)

type Registry struct {
	defs map[string]config.ActionDef
}

type Action struct {
	Name string
	Def  config.ActionDef
}

type Result struct {
	Stdout, Stderr string
	ExitCode       int
}

func NewRegistry(defs map[string]config.ActionDef) *Registry {
	return &Registry{defs: defs}
}

func (r *Registry) Applicable(n *model.Node) []Action {
	out := make([]Action, 0, len(r.defs))
	for name, def := range r.defs {
		if !whenMatches(def.When, n) {
			continue
		}
		out = append(out, Action{Name: name, Def: def})
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Name < out[j].Name })
	return out
}

func whenMatches(w config.ActionWhen, n *model.Node) bool {
	for _, key := range w.HasFrontmatter {
		if _, ok := n.FM.Extra[key]; !ok {
			return false
		}
	}
	return true
}

var nonAlnum = regexp.MustCompile(`[^A-Za-z0-9]+`)

func envKey(s string) string {
	s = strings.ToUpper(s)
	return nonAlnum.ReplaceAllString(s, "_")
}

func buildEnv(n *model.Node) []string {
	env := []string{
		"CASCADE_TITLE=" + n.Title(),
		"CASCADE_PATH=" + n.Path,
		"CASCADE_STATUS=" + string(n.FM.Status),
		"CASCADE_TAGS=" + strings.Join(n.FM.Tags, " "),
		"CASCADE_BODY_FILE=" + filepath.Join(n.Path, "index.md"),
	}
	for k, v := range n.FM.Extra {
		env = append(env, fmt.Sprintf("CASCADE_FM_%s=%v", envKey(k), v))
	}
	return env
}

func (a Action) Run(n *model.Node) (Result, error) {
	cmd := exec.Command("sh", "-c", a.Def.Cmd)
	cmd.Env = append(cmd.Env, buildEnv(n)...)
	// Inherit PATH and HOME from process env so tools like `gh` resolve.
	for _, k := range []string{"PATH", "HOME", "USER", "SHELL", "LANG", "TERM"} {
		if v := getenv(k); v != "" {
			cmd.Env = append(cmd.Env, k+"="+v)
		}
	}
	if a.Def.Stdin == "body" {
		cmd.Stdin = strings.NewReader(n.Body)
	}
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	err := cmd.Run()
	res := Result{Stdout: stdout.String(), Stderr: stderr.String()}
	if exitErr, ok := err.(*exec.ExitError); ok {
		res.ExitCode = exitErr.ExitCode()
		return res, nil // non-zero exit isn't a Go-level error
	}
	if err != nil {
		return res, err
	}
	return res, nil
}

func getenv(k string) string {
	for _, e := range globalEnv() {
		if strings.HasPrefix(e, k+"=") {
			return e[len(k)+1:]
		}
	}
	return ""
}

// Indirected for testability.
var globalEnv = func() []string {
	return osEnviron()
}
