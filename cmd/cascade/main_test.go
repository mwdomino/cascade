package main

import (
	"os/exec"
	"strings"
	"testing"
)

func TestVersionFlag(t *testing.T) {
	cmd := exec.Command("go", "run", ".", "--version")
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("run failed: %v\noutput: %s", err, out)
	}
	if !strings.Contains(string(out), "cascade") {
		t.Fatalf("expected 'cascade' in output, got: %s", out)
	}
}
