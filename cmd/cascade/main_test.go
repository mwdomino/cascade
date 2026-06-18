package main

import (
	"os/exec"
	"strings"
	"testing"
)

func TestVersionVariants(t *testing.T) {
	for _, arg := range []string{"version", "-v", "-version", "--version"} {
		t.Run(arg, func(t *testing.T) {
			cmd := exec.Command("go", "run", ".", arg)
			out, err := cmd.CombinedOutput()
			if err != nil {
				t.Fatalf("run failed: %v\noutput: %s", err, out)
			}
			if !strings.HasPrefix(strings.TrimSpace(string(out)), "cascade ") {
				t.Fatalf("expected output to start with 'cascade ', got: %s", out)
			}
			// `go run` builds an ephemeral binary without vcs metadata, so
			// `dev` is the legitimate fallback here. Release builds use
			// goreleaser's ldflag injection; `go install path@vX.Y.Z` uses
			// debug.BuildInfo.Main.Version. Both are exercised manually.
		})
	}
}

func TestHelpVariants(t *testing.T) {
	for _, arg := range []string{"help", "-h", "--help"} {
		t.Run(arg, func(t *testing.T) {
			cmd := exec.Command("go", "run", ".", arg)
			out, _ := cmd.CombinedOutput()
			if !strings.Contains(string(out), "launch the TUI") {
				t.Errorf("expected usage in output, got: %s", out)
			}
		})
	}
}
